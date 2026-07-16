@echo off
REM Windows Proxmark3 CLI wrapper for Keweenaw Endurance hardware tests.
REM Uses ProxSpace-built client against the Iceman Proxmark on COM3 by default.

setlocal
set "PM3_ROOT=C:\Users\gener\sdk\ProxSpace"
set "PM3_EXE=%PM3_ROOT%\pm3\proxmark3\client\proxmark3.exe"
set "MINGW_BIN=%PM3_ROOT%\msys2\mingw64\bin"
set "PATH=%MINGW_BIN%;%PATH%"

if not exist "%PM3_EXE%" (
  echo proxmark3.exe not found at %PM3_EXE%
  echo Build with ProxSpace first.
  exit /b 1
)

if "%PROXMARK3_PORT%"=="" set "PROXMARK3_PORT=COM3"

REM Go's CLIProxmarkReader already passes -p <port>; don't duplicate it
REM (proxmark3.exe errors: "got COM3 as port and now we got also: COM3").
echo %*| findstr /I /C:"-p " >nul
if %ERRORLEVEL%==0 (
  "%PM3_EXE%" %*
) else (
  "%PM3_EXE%" -p %PROXMARK3_PORT% -f %*
)
exit /b %ERRORLEVEL%
