@echo off
cd /d "%~dp0"
go run ./tools/release %*
pause
