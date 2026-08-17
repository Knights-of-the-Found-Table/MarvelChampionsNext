@echo off
rem One-click dev startup: Go backend (:3000) + Vite frontend (:8080).
cd /d "%~dp0"

where go >nul 2>&1 || (echo error: go not found in PATH & exit /b 1)
where npm >nul 2>&1 || (echo error: npm not found in PATH & exit /b 1)

if not exist web\node_modules (
  echo ==^> installing frontend dependencies
  pushd web
  call npm install --no-fund --no-audit
  popd
)

echo ==^> starting backend on :3000
rem Fixed dev secret so backend restarts don't invalidate login tokens.
if not defined MC_JWT_SECRET set "MC_JWT_SECRET=mcdev-insecure-jwt-secret"
start "mcg-backend" cmd /k go run ./cmd/server

echo ==^> starting frontend on :8080
pushd web
start "mcg-frontend" cmd /k npm run dev
popd

echo ==^> ready: http://localhost:8080
echo Close the two dev windows to stop the servers.
