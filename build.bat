@echo off
cd /d "%~dp0"
go run github.com/ajkula/ajkurig/cmd/release %*
pause
