import requests
import pytest

BASE_URL = "http://localhost:8081"
def test_get_catalog_apple():
    endpoint = "/v1/products"
    params = {"q":  "iphone","brand": "apple", "page_size": 10}
    url =f"{BASE_URL}{endpoint}"
    response = requests.get(url, params=params)
    assert response.status_code == 200
    data = response.json()
    assert "products" in data, "Ключ products не найден в ответе"

