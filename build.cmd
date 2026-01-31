set exename=iloviewer
REM set GOARCH=386
go env GOOS GOARCH
REM go build -ldflags "-s -w"
for /f "delims=" %%I in ('dir /b /ad "C:\Program Files (x86)\Windows Kits\10\Include"') do (
	set WINSDKVER=%%I
	goto :gotver
)
:gotver
for %%I in ("C:\Program Files (x86)\Windows Kits") do set "WINSDKROOT=%%~sI"
set "CGO_CFLAGS=-I%WINSDKROOT%\10\Include\%WINSDKVER%\winrt"
set "CGO_CXXFLAGS=-I%WINSDKROOT%\10\Include\%WINSDKVER%\winrt"
set "CGO_LDFLAGS="
go build -ldflags="-s -w" -o %exename%.exe
REM go build -ldflags="-H windowsgui -s -w" -o %exename%.exe
REM upx --best --lzma %exename%.exe
REM go build -compiler gccgo -gccgoflags "-s -w" 
pause