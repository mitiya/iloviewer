package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	webview "github.com/webview/webview_go"
)

func main() {
	urlFlag := flag.String("url", "", "Base URL, e.g. https://example.com/")
	loginFlag := flag.String("login", "", "Login username")
	passwordFlag := flag.String("password", "", "Login password")
	discUploadsFlag := flag.String("discurls", "", "Semicolon-separated URLs for disc_upload input")
	isTempCopy := flag.Bool("_tempcopy", false, "Internal flag - running from temp copy")
	debugFlag := flag.Bool("debug", false, "Run with console window for debugging")
	isDetached := flag.Bool("_detached", false, "Internal flag - already detached from console")
	flag.Parse()

	if *urlFlag == "" {
		fmt.Println("Usage example:")
		fmt.Println(`iloviewer.exe -url https://example.com -login username -password pass -discurls "url1;url2"`)
		fmt.Println("  iloviewer -debug -url ... (run with console for debugging)")
		return
	}

	// If not debug mode and not already detached, relaunch detached from console
	if !*debugFlag && !*isDetached && !*isTempCopy {
		relaunchDetached()
		return
	}

	// Cleanup old orphaned temp folders from previous runs
	cleanupOldTempFolders()

	// If not running from temp copy, create temp copy, wait for it, then cleanup
	if !*isTempCopy {
		launchTempCopyAndWait(*debugFlag)
		return
	}

	// Running from temp copy - just run the app, parent will cleanup

	// Update these if the server requires form login.
	login := *loginFlag
	password := *passwordFlag
	baseURL := *urlFlag

	var discUploads []string
	for _, v := range strings.Split(*discUploadsFlag, ";") {
		v = strings.TrimSpace(v)
		if v != "" {
			discUploads = append(discUploads, v)
		}
	}
	discUploadsJSON, _ := json.Marshal(discUploads)

	url := baseURL
	if login != "" || password != "" {
		if strings.HasPrefix(baseURL, "https://") {
			url = "https://" + login + ":" + password + "@" + strings.TrimPrefix(baseURL, "https://")
		} else if strings.HasPrefix(baseURL, "http://") {
			url = "http://" + login + ":" + password + "@" + strings.TrimPrefix(baseURL, "http://")
		}
	}

	// Create unique user data folder for isolation based on URL
	exeDir, _ := os.Executable()
	exeDir = filepath.Dir(exeDir)
	userDataDir := filepath.Join(exeDir, "webview_data")
	os.MkdirAll(userDataDir, 0755)

	// Generate unique ID based on URL to isolate profiles
	hashBytes := []byte(baseURL)
	sessionID := hex.EncodeToString(hashBytes)
	if len(sessionID) > 16 {
		sessionID = sessionID[:16]
	}
	sessionDataDir := filepath.Join(userDataDir, sessionID)
	fmt.Printf("WebView2 data folder: %s\n", sessionDataDir)

	// Set WebView2 user data folder via additional arguments
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--user-data-dir=\""+sessionDataDir+"\" --ignore-certificate-errors")

	w := webview.New(true)
	if w == nil {
		return
	}
	defer w.Destroy()

	w.SetTitle("iloviewer - " + baseURL)
	w.SetSize(1024, 768, webview.Hint(webview.HintNone))

	// Auto-fill login form (adjust selectors if needed).
	if login != "" || password != "" {
		js := `
(() => {
	const login = ` + "`" + login + "`" + `;
	const password = ` + "`" + password + "`" + `;
	const discOptions = ` + string(discUploadsJSON) + `;
	let clicked = false;
	let dropdown;
	let dropdownVisible = false;
	const LOGIN_KEY = 'iloviewer_autologin_last';

	function findInput(selectors) {
		for (const s of selectors) {
			const el = document.querySelector(s);
			if (el) return el;
		}
		return null;
	}

	function fireInputEvents(el) {
		el.dispatchEvent(new Event('input', { bubbles: true }));
		el.dispatchEvent(new Event('change', { bubbles: true }));
		el.dispatchEvent(new KeyboardEvent('keyup', { bubbles: true }));
	}

	function canAttemptLogin() {
		const last = Number(sessionStorage.getItem(LOGIN_KEY) || 0);
		return (Date.now() - last) > 15000;
	}

	function markLoginAttempt() {
		sessionStorage.setItem(LOGIN_KEY, String(Date.now()));
	}

	function ensureDropdown() {
		if (dropdown) return dropdown;
		dropdown = document.createElement('div');
		dropdown.style.position = 'absolute';
		dropdown.style.zIndex = '2147483647';
		dropdown.style.background = '#fff';
		dropdown.style.border = '1px solid #ccc';
		dropdown.style.boxShadow = '0 2px 6px rgba(0,0,0,0.2)';
		dropdown.style.maxHeight = '200px';
		dropdown.style.overflowY = 'auto';
		dropdown.style.fontSize = '12px';
		dropdown.style.display = 'none';
		document.body.appendChild(dropdown);
		return dropdown;
	}

	function showDropdown(input, options) {
		const dd = ensureDropdown();
		dropdownVisible = true;
		const rect = input.getBoundingClientRect();
		dd.style.left = (rect.left + window.scrollX) + 'px';
		dd.style.top = (rect.bottom + window.scrollY) + 'px';
		dd.style.width = rect.width + 'px';
		dd.innerHTML = '';
		for (const opt of options) {
			const item = document.createElement('div');
			item.textContent = opt;
			item.style.padding = '6px 8px';
			item.style.cursor = 'pointer';
			item.addEventListener('mousedown', (e) => {
				e.preventDefault();
				input.value = opt;
				fireInputEvents(input);
				hideDropdown();
			});
			dd.appendChild(item);
		}
		dd.style.display = options.length ? 'block' : 'none';
	}

	function hideDropdown() {
		if (!dropdown) return;
		dropdownVisible = false;
		dropdown.style.display = 'none';
	}

	function attachDiscDropdown(input) {
		if (!input || !Array.isArray(discOptions) || !discOptions.length) return;
		if (input.__discDropdownAttached) return;
		input.__discDropdownAttached = true;
		input.addEventListener('focus', () => {
			showDropdown(input, discOptions);
		});
		input.addEventListener('input', () => {
			const q = input.value.toLowerCase();
			const filtered = discOptions.filter(o => o.toLowerCase().includes(q));
			showDropdown(input, filtered);
		});
		input.addEventListener('contextmenu', (e) => {
			e.preventDefault();
			showDropdown(input, discOptions);
		});
		input.addEventListener('blur', () => {
			setTimeout(hideDropdown, 150);
		});
		window.addEventListener('scroll', () => {
			if (dropdownVisible) showDropdown(input, discOptions);
		}, true);
		document.addEventListener('mousedown', (e) => {
			if (dropdownVisible && dropdown && !dropdown.contains(e.target) && e.target !== input) {
				hideDropdown();
			}
		});
	}

	function fillOnce() {
		const userInput = findInput([
			'input[name*="user" i]',
			'input[name*="login" i]',
			'input[name*="email" i]',
			'input[id*="user" i]',
			'input[id*="login" i]',
			'input[id*="email" i]',
			'input[type="text"]',
			'input[type="email"]'
		]);
		const passInput = findInput([
			'input[name*="pass" i]',
			'input[id*="pass" i]',
			'input[type="password"]'
		]);
		const discInput = document.getElementById('disc_upload');

		// Fill inputs when they appear
		if (userInput && login && userInput.value !== login) {
			userInput.value = login;
			fireInputEvents(userInput);
		}
		if (passInput && password && passInput.value !== password) {
			passInput.value = password;
			fireInputEvents(passInput);
		}
		if (discInput) {
			attachDiscDropdown(discInput);
			if (Array.isArray(discOptions) && discOptions.length && !discInput.value) {
				discInput.value = discOptions[0];
				fireInputEvents(discInput);
			}
		}

		// Try to click button only after fields are filled
		if (!clicked && (userInput || passInput)) {
			const btn = document.getElementById('ID_LOGON') || document.querySelector('button[type="submit"], input[type="submit"]');
			if (btn) {
				if (!canAttemptLogin()) {
					return false;
				}
				if (!btn.disabled) {
					clicked = true;
					markLoginAttempt();
					btn.click();
					return true;
				}
			}
		}

		return false;
	}

	const start = Date.now();
	const timer = setInterval(() => {
		if (fillOnce() || Date.now() - start > 15000) {
			clearInterval(timer);
		}
	}, 500);

	// Update title with current URL
	let lastUrl = '';
	setInterval(() => {
		const currentUrl = window.location.href;
		if (currentUrl !== lastUrl) {
			lastUrl = currentUrl;
			document.title = 'iloviewer - ' + currentUrl;
		}
	}, 1000);
})();
`
		w.Init(js)
	}

	w.Navigate(url)
	w.Run()
}

func relaunchDetached() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}

	args := []string{"-_detached"}
	args = append(args, os.Args[1:]...)

	cmd := exec.Command(exePath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	cmd.Start()
}

func launchTempCopyAndWait(debug bool) {
	// Get current exe path
	exePath, err := os.Executable()
	if err != nil {
		if debug {
			fmt.Printf("Error getting exe path: %v\n", err)
		}
		return
	}

	// Create unique exe name using random bytes for true uniqueness
	tempDir := os.TempDir()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	profileID := hex.EncodeToString(randomBytes)

	tempName := "iloviewer_" + profileID + ".exe"
	tempExePath := filepath.Join(tempDir, tempName)

	// Profile folder path that WebView2 will create in AppData
	// WebView2 uses the FULL exe name including .exe as folder name!
	appData := os.Getenv("APPDATA")
	appDataProfile := filepath.Join(appData, tempName)

	if debug {
		fmt.Printf("=== iloviewer launcher ===\n")
		fmt.Printf("Temp exe: %s\n", tempExePath)
		fmt.Printf("Profile folder (expected): %s\n", appDataProfile)
		fmt.Printf("Copying exe...\n")
	}

	// Copy exe to temp
	src, err := os.Open(exePath)
	if err != nil {
		if debug {
			fmt.Printf("Error opening exe: %v\n", err)
		}
		return
	}
	defer src.Close()

	dst, err := os.Create(tempExePath)
	if err != nil {
		if debug {
			fmt.Printf("Error creating temp exe: %v\n", err)
		}
		return
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		if debug {
			fmt.Printf("Error copying exe: %v\n", err)
		}
		return
	}
	dst.Close()
	if debug {
		fmt.Printf("Exe copied successfully\n")
	}

	// Prepare command line args
	args := []string{"-_tempcopy"}
	args = append(args, os.Args[1:]...)

	if debug {
		fmt.Printf("Launching child process...\n")
	}

	// Launch temp copy and WAIT for it to finish
	cmd := exec.Command(tempExePath, args...)
	err = cmd.Run() // This blocks until child process exits

	if debug {
		fmt.Printf("Child process exited (err=%v)\n", err)
		fmt.Printf("Waiting 3 seconds for WebView2 to release files...\n")
	}

	// Wait for WebView2 to release files
	time.Sleep(3 * time.Second)

	// Check if profile folder exists
	if _, statErr := os.Stat(appDataProfile); statErr == nil {
		if debug {
			fmt.Printf("Profile folder exists, deleting: %s\n", appDataProfile)
		}
		err = os.RemoveAll(appDataProfile)
		if debug {
			if err != nil {
				fmt.Printf("Error deleting profile folder: %v\n", err)
			} else {
				fmt.Printf("Profile folder deleted successfully\n")
			}
		}
	} else {
		if debug {
			fmt.Printf("Profile folder does not exist: %s\n", appDataProfile)
		}
	}

	// Check if temp exe exists and delete it
	if _, statErr := os.Stat(tempExePath); statErr == nil {
		if debug {
			fmt.Printf("Deleting temp exe: %s\n", tempExePath)
		}
		err = os.Remove(tempExePath)
		if debug {
			if err != nil {
				fmt.Printf("Error deleting temp exe: %v\n", err)
			} else {
				fmt.Printf("Temp exe deleted successfully\n")
			}
		}
	} else {
		if debug {
			fmt.Printf("Temp exe does not exist: %s\n", tempExePath)
		}
	}

	if debug {
		fmt.Printf("=== Cleanup complete ===\n")
	}
}

func cleanupOldTempFolders() {
	// Remove orphaned profile folders and exe files older than 1 hour
	tempDir := os.TempDir()
	cutoff := time.Now().Add(-1 * time.Hour)

	// Cleanup old profile folders
	matches, _ := filepath.Glob(filepath.Join(tempDir, "iloviewer_profile_*"))
	for _, path := range matches {
		if info, err := os.Stat(path); err == nil {
			if info.ModTime().Before(cutoff) {
				os.RemoveAll(path)
			}
		}
	}

	// Cleanup old temp exe files
	matches, _ = filepath.Glob(filepath.Join(tempDir, "iloviewer_*.exe"))
	for _, path := range matches {
		if info, err := os.Stat(path); err == nil {
			if info.ModTime().Before(cutoff) {
				os.Remove(path)
			}
		}
	}

	// Cleanup AppData profiles
	appData := os.Getenv("APPDATA")
	if appData != "" {
		matches, _ := filepath.Glob(filepath.Join(appData, "iloviewer_*"))
		for _, path := range matches {
			if info, err := os.Stat(path); err == nil {
				if info.ModTime().Before(cutoff) {
					os.RemoveAll(path)
				}
			}
		}
	}
}
