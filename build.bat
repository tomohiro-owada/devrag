@echo off
echo Building markdown-vector-mcp...

if not exist bin mkdir bin

set LDFLAGS=-s -w
set TAGS=netgo

REM Version info
if "%VERSION%"=="" set VERSION=1.0.0

echo Version: %VERSION%
echo.

echo Building for Windows (amd64)...
set GOOS=windows
set GOARCH=amd64
go build -tags %TAGS% -ldflags="%LDFLAGS% -X main.Version=%VERSION%" -o bin\markdown-vector-mcp-windows-amd64.exe cmd\main.go

echo.
echo Build complete!
dir bin\
