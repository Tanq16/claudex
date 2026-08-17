# Popup Template

Complete popup implementation with Catppuccin Mocha theme.

---

## popup.html

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>[EXTENSION_NAME]</title>
  <link rel="stylesheet" href="popup.css">
</head>
<body>
  <div class="container">
    <!-- Header -->
    <header class="header">
      <h1>[EXTENSION_NAME]</h1>
      <p class="subtitle">Brief tagline here</p>
    </header>

    <!-- Main Content -->
    <main class="content">
      <!-- Example: Input Section -->
      <div class="section">
        <label for="input-field">Label</label>
        <input type="text" id="input-field" placeholder="Enter value...">
      </div>

      <!-- Example: Toggle Section -->
      <div class="section row">
        <span>Enable Feature</span>
        <label class="toggle">
          <input type="checkbox" id="toggle-feature">
          <span class="slider"></span>
        </label>
      </div>

      <!-- Actions -->
      <div class="actions">
        <button id="action-btn" class="primary">Primary Action</button>
        <button id="secondary-btn">Secondary</button>
      </div>
    </main>

    <!-- Status/Output -->
    <footer class="footer">
      <div id="status" class="status"></div>
    </footer>
  </div>

  <script src="popup.js"></script>
</body>
</html>
```

---

## popup.css

```css
/* =============================================================================
   Catppuccin Mocha Theme
   ============================================================================= */

:root {
  /* Catppuccin Mocha */
  --rosewater: #f5e0dc;
  --flamingo: #f2cdcd;
  --pink: #f5c2e7;
  --mauve: #cba6f7;
  --red: #f38ba8;
  --maroon: #eba0ac;
  --peach: #fab387;
  --yellow: #f9e2af;
  --green: #a6e3a1;
  --teal: #94e2d5;
  --sky: #89dceb;
  --sapphire: #74c7ec;
  --blue: #89b4fa;
  --lavender: #b4befe;
  --text: #cdd6f4;
  --subtext1: #bac2de;
  --subtext0: #a6adc8;
  --overlay2: #9399b2;
  --overlay1: #7f849c;
  --overlay0: #6c7086;
  --surface2: #585b70;
  --surface1: #45475a;
  --surface0: #313244;
  --base: #1e1e2e;
  --mantle: #181825;
  --crust: #11111b;
}

/* =============================================================================
   Reset & Base
   ============================================================================= */

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 
               'Helvetica Neue', Arial, sans-serif;
  font-size: 14px;
  line-height: 1.5;
  background-color: var(--base);
  color: var(--text);
  min-width: 320px;
  max-width: 400px;
}

/* =============================================================================
   Layout
   ============================================================================= */

.container {
  padding: 16px;
}

.header {
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--surface1);
}

.header h1 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 4px;
}

.header .subtitle {
  font-size: 12px;
  color: var(--subtext0);
}

.content {
  margin-bottom: 16px;
}

.section {
  margin-bottom: 12px;
}

.section.row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.section label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--subtext1);
  margin-bottom: 6px;
}

.actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}

.footer {
  padding-top: 12px;
  border-top: 1px solid var(--surface1);
}

/* =============================================================================
   Form Elements
   ============================================================================= */

input[type="text"],
input[type="number"],
input[type="url"],
textarea {
  width: 100%;
  padding: 8px 12px;
  font-size: 14px;
  background-color: var(--surface0);
  color: var(--text);
  border: 1px solid var(--surface1);
  border-radius: 6px;
  transition: border-color 0.2s;
}

input:focus,
textarea:focus {
  outline: none;
  border-color: var(--blue);
}

input::placeholder,
textarea::placeholder {
  color: var(--overlay0);
}

/* =============================================================================
   Buttons
   ============================================================================= */

button {
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 500;
  background-color: var(--surface0);
  color: var(--text);
  border: 1px solid var(--surface1);
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.2s, border-color 0.2s;
}

button:hover {
  background-color: var(--surface1);
}

button:active {
  background-color: var(--surface2);
}

button.primary {
  background-color: var(--blue);
  color: var(--crust);
  border: none;
  flex: 1;
}

button.primary:hover {
  background-color: var(--sapphire);
}

button.primary:active {
  background-color: var(--sky);
}

button.danger {
  background-color: var(--red);
  color: var(--crust);
  border: none;
}

button.danger:hover {
  background-color: var(--maroon);
}

button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* =============================================================================
   Toggle Switch
   ============================================================================= */

.toggle {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle .slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--surface1);
  border-radius: 24px;
  transition: 0.2s;
}

.toggle .slider:before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: var(--text);
  border-radius: 50%;
  transition: 0.2s;
}

.toggle input:checked + .slider {
  background-color: var(--blue);
}

.toggle input:checked + .slider:before {
  transform: translateX(20px);
}

/* =============================================================================
   Status Messages
   ============================================================================= */

.status {
  font-size: 12px;
  padding: 8px;
  border-radius: 6px;
  text-align: center;
}

.status:empty {
  display: none;
}

.status.success {
  background-color: rgba(166, 227, 161, 0.1);
  color: var(--green);
}

.status.error {
  background-color: rgba(243, 139, 168, 0.1);
  color: var(--red);
}

.status.warning {
  background-color: rgba(249, 226, 175, 0.1);
  color: var(--yellow);
}

.status.info {
  background-color: rgba(137, 180, 250, 0.1);
  color: var(--blue);
}

/* =============================================================================
   Utility Classes
   ============================================================================= */

.hidden {
  display: none !important;
}

.text-muted {
  color: var(--subtext0);
}

.text-small {
  font-size: 12px;
}

.mt-8 { margin-top: 8px; }
.mt-16 { margin-top: 16px; }
.mb-8 { margin-bottom: 8px; }
.mb-16 { margin-bottom: 16px; }
```

---

## popup.js

```javascript
// =============================================================================
// Popup Script
// =============================================================================

document.addEventListener('DOMContentLoaded', init);

async function init() {
  // Load saved settings
  const settings = await loadSettings();
  applySettings(settings);

  // Setup event listeners
  setupEventListeners();
}

// =============================================================================
// Settings
// =============================================================================

async function loadSettings() {
  const result = await chrome.storage.local.get('settings');
  return result.settings || {
    enabled: true,
    // Add default settings here
  };
}

async function saveSettings(settings) {
  await chrome.storage.local.set({ settings });
}

function applySettings(settings) {
  const toggle = document.getElementById('toggle-feature');
  if (toggle) {
    toggle.checked = settings.enabled;
  }
}

// =============================================================================
// Event Listeners
// =============================================================================

function setupEventListeners() {
  // Primary action button
  const actionBtn = document.getElementById('action-btn');
  if (actionBtn) {
    actionBtn.addEventListener('click', handlePrimaryAction);
  }

  // Secondary button
  const secondaryBtn = document.getElementById('secondary-btn');
  if (secondaryBtn) {
    secondaryBtn.addEventListener('click', handleSecondaryAction);
  }

  // Toggle
  const toggle = document.getElementById('toggle-feature');
  if (toggle) {
    toggle.addEventListener('change', handleToggleChange);
  }
}

// =============================================================================
// Action Handlers
// =============================================================================

async function handlePrimaryAction() {
  const status = document.getElementById('status');
  
  try {
    showStatus('Processing...', 'info');
    
    // Get current tab
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    
    // Execute script in page context
    const results = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: pageFunction
    });
    
    const result = results[0].result;
    showStatus(`Success: ${result}`, 'success');
    
  } catch (error) {
    showStatus(`Error: ${error.message}`, 'error');
  }
}

// Function that runs in page context
function pageFunction() {
  // This code runs in the web page, not the extension
  return document.title;
}

async function handleSecondaryAction() {
  showStatus('Secondary action clicked', 'info');
}

async function handleToggleChange(event) {
  const settings = await loadSettings();
  settings.enabled = event.target.checked;
  await saveSettings(settings);
  
  showStatus(settings.enabled ? 'Enabled' : 'Disabled', 'info');
}

// =============================================================================
// UI Helpers
// =============================================================================

function showStatus(message, type = 'info') {
  const status = document.getElementById('status');
  if (status) {
    status.textContent = message;
    status.className = `status ${type}`;
  }
}

function clearStatus() {
  const status = document.getElementById('status');
  if (status) {
    status.textContent = '';
    status.className = 'status';
  }
}

// =============================================================================
// Message Passing (to content script or background)
// =============================================================================

async function sendToContentScript(action, data = {}) {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  return chrome.tabs.sendMessage(tab.id, { action, ...data });
}

async function sendToBackground(action, data = {}) {
  return chrome.runtime.sendMessage({ action, ...data });
}
```

---

## Placeholders

| Placeholder | Replace With |
|-------------|--------------|
| `[EXTENSION_NAME]` | Your extension name (e.g., "Cookie Extractor") |
