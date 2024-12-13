@echo off
echo Available COM ports:
powershell "Get-WmiObject Win32_SerialPort | Select-Object Name, DeviceID"
echo.

set /p COM_PORT="Enter COM port (example COM23): "

echo Starting ComTcpBridge on %COM_PORT%...
ComTcpBridge.exe -port %COM_PORT%
pause