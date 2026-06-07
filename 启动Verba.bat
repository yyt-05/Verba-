@echo off
chcp 65001 >nul
echo 正在启动 Verba AI 同声传译...
echo.

:: 启动后端
cd /d "%~dp0server"
start "Verba Server" /MIN .\verba.exe

:: 等后端就绪
timeout /t 2 /nobreak >nul

:: 启动客户端 (Release 版，无需 Flutter 环境)
cd /d "%~dp0client\build\windows\x64\runner\Release"
start "Verba" .\verba_app.exe

echo Verba 已启动！悬浮窗应该已出现在屏幕上。
echo 按任意键关闭此窗口...
pause >nul
