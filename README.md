# iloviewer

Просмотрщик веб-интерфейсов с автологином на базе WebView2.

## Зависимости

Приложение зависит от **Microsoft Edge WebView2 Runtime**, который должен быть установлен в системе.

### Проверка установки WebView2 Runtime

В большинстве современных Windows 10/11 WebView2 Runtime уже установлен. Проверить можно:
- Открыть `Параметры` → `Приложения` → найти "Microsoft Edge WebView2 Runtime"

### Установка WebView2 Runtime

Если не установлен, скачайте с официального сайта:
https://developer.microsoft.com/microsoft-edge/webview2/#download-section

Есть два варианта:
1. **Evergreen Runtime** (рекомендуется) - ~150 KB, автообновляется системой
2. **Fixed Version Runtime** - для портабельного использования (~180 MB)

## Портабельность

### Вариант 1: Установить Runtime на целевой системе
Самый простой - установить Evergreen Runtime на всех компьютерах.

### Вариант 2: Fixed Version Runtime (полностью портабельный)
1. Скачайте Fixed Version Runtime для вашей архитектуры
2. Распакуйте рядом с exe в папку, например `webview2_runtime`
3. При запуске укажите переменную окружения:
   ```
   set WEBVIEW2_BROWSER_EXECUTABLE_FOLDER=.\webview2_runtime
   iloviewer.exe -url https://...
   ```

## Использование

```bash
iloviewer.exe -url https://example.com -login username -password pass -discurls "url1;url2"
```

### Параметры:
- `-url` - базовый URL (обязательно)
- `-login` - логин для автозахода
- `-password` - пароль для автозахода
- `-discurls` - список URL через точку с запятой для поля disc_upload

### Изоляция экземпляров

Каждый запуск приложения создаёт уникальную папку профиля WebView2, что позволяет:
- Открывать несколько копий с разными логинами одновременно
- Избежать конфликтов сессий и разлогонов
- После закрытия временные данные удаляются

## Сборка

```bash
go build -ldflags "-s -w" -o iloviewer.exe
```

Для минимального размера можно использовать UPX:
```bash
upx --best --lzma iloviewer.exe
```
