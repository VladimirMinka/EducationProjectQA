package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	pbCart "awesomeProject/gen/store/api/cart/v1"
	pbCatalog "awesomeProject/gen/store/api/catalog/v1"
	pbDelivery "awesomeProject/gen/store/api/delivery/v1"
	pbOrder "awesomeProject/gen/store/api/order/v1"
	pbPromo "awesomeProject/gen/store/api/promo/v1"
	pbUser "awesomeProject/gen/store/api/user/v1"

	"awesomeProject/internal/auth"
	"awesomeProject/internal/handler"
	"awesomeProject/internal/logging"
	"awesomeProject/internal/repository/postgres"
	"awesomeProject/internal/service"
	"awesomeProject/internal/worker"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://store:store@localhost:5432/store?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-me"
	}
	jwtManager := auth.NewManager(jwtSecret, 24*time.Hour)

	db, err := postgres.Open(dsn)
	if err != nil {
		logging.Logger.Error("database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	userRepo := postgres.NewUserRepository(db)
	userService := service.NewUserService(userRepo, jwtManager)
	userHandler := handler.NewUserHandler(userService)

	catalogRepo := postgres.NewCatalogRepository(db)
	catalogService := service.NewCatalogService(catalogRepo)
	catalogHandler := handler.NewCatalogHandler(catalogService)

	promoRepo := postgres.NewPromoRepository(db)
	promoService := service.NewPromoService(promoRepo)
	promoHandler := handler.NewPromoHandler(promoService)

	cartRepo := postgres.NewCartRepository(db)
	cartService := service.NewCartService(cartRepo, catalogService, userService, promoService)
	cartHandler := handler.NewCartHandler(cartService)

	deliveryRepo := postgres.NewDeliveryRepository(db)
	deliveryService := service.NewDeliveryService(deliveryRepo, userService)
	deliveryHandler := handler.NewDeliveryHandler(deliveryService)

	orderRepo := postgres.NewOrderRepository(db)
	orderJobRepo := postgres.NewOrderJobRepository(db)
	orderService := service.NewOrderService(orderRepo, orderJobRepo, cartService, catalogService, userService, deliveryService)
	orderHandler := handler.NewOrderHandler(orderService)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go worker.NewOrderStatusWorker(orderRepo, orderJobRepo, orderService.StatusDelay()).Run(workerCtx)

	// Listen on all interfaces; dial via explicit IPv4 loopback (see GRPC_ENDPOINT).
	grpcListen := ":50051"
	lis, err := net.Listen("tcp", grpcListen)
	if err != nil {
		logging.Logger.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			auth.UnaryLoggingInterceptor(),
			auth.UnaryInterceptor(jwtManager),
		),
	)

	pbUser.RegisterUserServiceServer(grpcServer, userHandler)
	pbCatalog.RegisterCatalogServiceServer(grpcServer, catalogHandler)
	pbCart.RegisterCartServiceServer(grpcServer, cartHandler)
	pbOrder.RegisterOrderServiceServer(grpcServer, orderHandler)
	pbPromo.RegisterPromoServiceServer(grpcServer, promoHandler)
	pbDelivery.RegisterDeliveryServiceServer(grpcServer, deliveryHandler)

	reflection.Register(grpcServer)

	go func() {
		logging.Logger.Info("starting gRPC", "addr", grpcListen)
		if err := grpcServer.Serve(lis); err != nil {
			logging.Logger.Error("failed to serve gRPC", "error", err)
			os.Exit(1)
		}
	}()

	grpcDial := os.Getenv("GRPC_ENDPOINT")
	if grpcDial == "" {
		// Не `:50051`: NewClient лениво резолвит и в Alpine часто сначала ::1 (~8s hang).
		grpcDial = "127.0.0.1:50051"
	}

	httpPort := ":8080"
	if err := runHTTPGateway(context.Background(), grpcDial, httpPort); err != nil {
		logging.Logger.Error("failed to serve HTTP gateway", "error", err)
		os.Exit(1)
	}
}

func runHTTPGateway(ctx context.Context, grpcEndpoint, httpPort string) error {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Один долгоживущий conn на все сервисы: иначе каждый FromEndpoint
	// лениво диалит свой канал → первый hit в cart и orders по ~8s каждый.
	conn, err := grpc.NewClient(grpcEndpoint, opts...)
	if err != nil {
		return fmt.Errorf("dial grpc %s: %w", grpcEndpoint, err)
	}
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	if err := pbUser.RegisterUserServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register user gateway: %w", err)
	}
	if err := pbCatalog.RegisterCatalogServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register catalog gateway: %w", err)
	}
	if err := pbCart.RegisterCartServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register cart gateway: %w", err)
	}
	if err := pbOrder.RegisterOrderServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register order gateway: %w", err)
	}
	if err := pbPromo.RegisterPromoServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register promo gateway: %w", err)
	}
	if err := pbDelivery.RegisterDeliveryServiceHandler(ctx, mux, conn); err != nil {
		return fmt.Errorf("register delivery gateway: %w", err)
	}

	warmCtx, warmCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer warmCancel()
	if err := warmGateway(warmCtx, conn); err != nil {
		logging.Logger.Warn("warmup warning", "error", err)
	} else {
		logging.Logger.Info("warmup ok", "grpc", grpcEndpoint)
	}

	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		webDir = "web/dist"
	}

	logging.Logger.Info("starting HTTP gateway + UI", "addr", httpPort, "web", webDir, "grpc", grpcEndpoint)
	// outermost: logs CORS OPTIONS and all API/UI traffic
	return http.ListenAndServe(httpPort, withAccessLog(withCORS(withGzip(withStaticCache(withUI(mux, webDir))))))
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// withAccessLog writes a short JSON access line. Full request/response bodies,
// user_id and role are logged on the gRPC interceptor (gateway → gRPC).
func withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logging.Logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// warmGateway устанавливает gRPC-канал и прогревает каталог/БД до первого пользовательского запроса.
func warmGateway(ctx context.Context, conn *grpc.ClientConn) error {
	client := pbCatalog.NewCatalogServiceClient(conn)
	var last error
	for attempt := 0; attempt < 20; attempt++ {
		if ctx.Err() != nil {
			return fmt.Errorf("warmup canceled: %w (last: %v)", ctx.Err(), last)
		}
		_, err := client.ListProducts(ctx, &pbCatalog.ListProductsRequest{})
		if err == nil {
			return nil
		}
		last = err
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("list products: %w", last)
}

func withUI(api http.Handler, webDir string) http.Handler {
	fs := http.Dir(webDir)
	fileServer := http.FileServer(fs)
	index := filepath.Join(webDir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			api.ServeHTTP(w, r)
			return
		}

		f, err := fs.Open(r.URL.Path)
		if err == nil {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !stat.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		http.ServeFile(w, r, index)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withStaticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/assets/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasPrefix(r.URL.Path, "/images/"):
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		next.ServeHTTP(w, r)
	})
}

func gzippable(path string) bool {
	switch {
	case strings.HasSuffix(path, ".png"),
		strings.HasSuffix(path, ".jpg"),
		strings.HasSuffix(path, ".jpeg"),
		strings.HasSuffix(path, ".webp"),
		strings.HasSuffix(path, ".gif"),
		strings.HasSuffix(path, ".woff2"),
		strings.HasSuffix(path, ".woff"):
		return false
	default:
		return true
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.gz.Write(b)
}

func withGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead ||
			!strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") ||
			!gzippable(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, gz: gz}, r)
	})
}
