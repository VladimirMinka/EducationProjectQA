#STORE-API SIMULATOR
-CURRENT PROJECT NEED FOR QA PRACTICE AND STUDYING MENTIES BEST PRACTICE
-RULES
# CORE ARCHITECTURE & STACK
1. STACK: Golang, Postgres, ELK, Vue
2. ARCHITECTURE: Modular Monolith. Create a new folder for each module: `src/{module_name}/`.
3. PATTERNS: Strictly follow Handler -> Service -> Repository layers.

# DEVELOPMENT RULES
4. NO HALLUCINATIONS: If requirements, data formats, or logic are unclear, stop and ask the user. Do not invent or assume requirements.
5. STEP-BY-STEP: Before writing complex logic, briefly output your plan in chat to get user approval.

# QA & EDUCATIONAL FEATURES (INTENTIONAL BUGS)
6. BUG INJECTION: If the user asks to introduce a bug, clarify the exact mechanics and specific impact of this bug first.
7. SECRET BUGS TRACKING: Every intentional bug MUST be documented in `BUGS.MD`. Ensure `BUGS.MD` is added to `.gitignore`.
8. NO CODE CLUES: NEVER add comments, FIXMEs, or descriptive variable names in the code that hint at the intentional bugs. The code must look perfectly normal.

# DOCUMENTATION
9. RELEASE NOTES: Document all new features in `RELEASE.MD`. Always include a section "IMPACT FOR QA" detailing what tests might break or need attention.
10. TECHNICAL DOCS: Update `DOCS.MD` with API specs, UI specs, and exposed ports for every new feature.
11. LANGUAGE: Write all content in `DOCS.MD` and `RELEASE.MD` in Russian.