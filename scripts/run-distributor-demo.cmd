@echo off
REM Обход ExecutionPolicy (скрипты .ps1 часто заблокированы в Windows по умолчанию)
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0run-distributor-demo.ps1" %*
