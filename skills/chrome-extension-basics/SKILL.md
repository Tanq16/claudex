---
name: chrome-extension-basics
description: Use when creating Chrome extensions - covers project structure, manifest.json, popup/content scripts, and Catppuccin color scheme
user-invocable: false
---

# Chrome Extension Basics

**Standardized patterns for Chrome extension development.**

## When to Use

Use this skill when:
- Creating a new Chrome extension
- Setting up extension project structure
- Implementing popup UI with Catppuccin theme
- Adding content scripts or background service workers

**Related skills:**
- `project-readme` - README templates (includes extension template with security disclaimer)
- `project-ci-cd` - Makefile for building extension zip

## Start here — required reading

Read this now, in full, before building the extension popup — it carries the complete popup implementation.

**Always:**
- `./references/popup-template.md` — complete popup HTML/CSS/JS with the Catppuccin theme

## Project Layout

```
extension-root/
├── manifest.json           # Extension configuration (required)
├── Makefile                # Build zip for distribution
├── README.md
├── .github/
│   ├── assets/
│   │   └── logo.png        # Project logo (128x128 or larger)
│   └── workflows/
│       └── release.yaml    # Automated releases
├── icons/
│   ├── icon16.png          # Toolbar icon
│   ├── icon32.png          # Windows icon
│   ├── icon48.png          # Extensions page
│   └── icon128.png         # Chrome Web Store / installation
├── popup/
│   ├── popup.html          # Popup UI (when extension clicked)
│   ├── popup.css           # Popup styles (Catppuccin)
│   └── popup.js            # Popup logic
├── content/
│   └── content.js          # Content script (runs in web pages)
├── background/
│   └── service-worker.js   # Background service worker (Manifest V3)
└── lib/                    # Shared libraries (optional)
    └── utils.js
```

**Key rules:**
- Use Manifest V3 (V2 is deprecated)
- Keep icons in dedicated `icons/` directory
- Separate popup, content, and background scripts into directories
- This skill assumes unpacked/sideloaded distribution (load unpacked); adjust packaging if targeting the Chrome Web Store

## Manifest V3 Template

```json
{
  "manifest_version": 3,
  "name": "[EXTENSION_NAME]",
  "version": "1.0.0",
  "description": "[Brief description of what the extension does]",
  
  "icons": {
    "16": "icons/icon16.png",
    "32": "icons/icon32.png",
    "48": "icons/icon48.png",
    "128": "icons/icon128.png"
  },
  
  "action": {
    "default_popup": "popup/popup.html",
    "default_icon": {
      "16": "icons/icon16.png",
      "32": "icons/icon32.png"
    }
  },
  
  "permissions": [
    "activeTab",
    "storage",
    "scripting"
  ],
  
  "background": {
    "service_worker": "background/service-worker.js"
  },
  
  "content_scripts": [
    {
      "matches": ["<all_urls>"],
      "js": ["content/content.js"],
      "run_at": "document_idle"
    }
  ]
}
```

### Common Permissions

| Permission | Purpose |
|------------|---------|
| `activeTab` | Access current tab when extension clicked (safest) |
| `storage` | Save settings using `chrome.storage` |
| `cookies` | Read/write cookies (requires host permissions) |
| `webRequest` | Monitor network requests |
| `tabs` | Access tab URLs and metadata |
| `scripting` | Programmatically inject scripts |

**Principle:** Request minimum permissions needed. Add `host_permissions` only for specific domains when possible.

## Color Scheme (Catppuccin Mocha)

Default new extensions to the Catppuccin Mocha dark theme; match the project's existing palette if one is already established:

```css
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

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background-color: var(--base);
  color: var(--text);
  min-width: 300px;
  padding: 16px;
}

button {
  background-color: var(--surface0);
  color: var(--text);
  border: 1px solid var(--surface1);
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.2s;
}

button:hover {
  background-color: var(--surface1);
}

button.primary {
  background-color: var(--blue);
  color: var(--crust);
  border: none;
}

button.primary:hover {
  background-color: var(--sapphire);
}

input, textarea {
  background-color: var(--surface0);
  color: var(--text);
  border: 1px solid var(--surface1);
  padding: 8px 12px;
  border-radius: 6px;
  width: 100%;
}

input:focus, textarea:focus {
  outline: none;
  border-color: var(--blue);
}

.success { color: var(--green); }
.error { color: var(--red); }
.warning { color: var(--yellow); }
.info { color: var(--blue); }
```

## Popup Template

See `./references/popup-template.md` for complete popup HTML/CSS/JS.

### Minimal popup.html

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
    <h1>[EXTENSION_NAME]</h1>
    <p class="description">Brief description here</p>
    
    <div class="actions">
      <button id="action-btn" class="primary">Do Something</button>
    </div>
    
    <div id="status" class="status"></div>
  </div>
  
  <script src="popup.js"></script>
</body>
</html>
```

### Minimal popup.js

```javascript
document.addEventListener('DOMContentLoaded', () => {
  const actionBtn = document.getElementById('action-btn');
  const status = document.getElementById('status');

  actionBtn.addEventListener('click', async () => {
    try {
      // Get current tab
      const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
      
      // Execute script in tab
      const results = await chrome.scripting.executeScript({
        target: { tabId: tab.id },
        func: () => {
          // This runs in the page context
          return document.title;
        }
      });
      
      status.textContent = `Page title: ${results[0].result}`;
      status.className = 'status success';
    } catch (error) {
      status.textContent = `Error: ${error.message}`;
      status.className = 'status error';
    }
  });
});
```

## Content Script Pattern

Content scripts run in web page context:

```javascript
// content/content.js

// Run when script loads
(function() {
  console.log('[ExtensionName] Content script loaded');
  
  // Listen for messages from popup or background
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === 'getData') {
      const data = extractData();
      sendResponse({ success: true, data });
    }
    return true; // Keep channel open for async response
  });
  
  function extractData() {
    // Extract data from page
    return {
      title: document.title,
      url: window.location.href
    };
  }
})();
```

## Background Service Worker

For persistent background tasks:

```javascript
// background/service-worker.js

// Extension installed
chrome.runtime.onInstalled.addListener((details) => {
  console.log('[ExtensionName] Installed:', details.reason);
  
  // Initialize storage with defaults
  chrome.storage.local.set({
    settings: {
      enabled: true
    }
  });
});

// Listen for messages
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'someAction') {
    handleAction(message.data)
      .then(result => sendResponse({ success: true, result }))
      .catch(error => sendResponse({ success: false, error: error.message }));
    return true; // Async response
  }
});

async function handleAction(data) {
  // Perform background task
  return { processed: true };
}
```

## Storage Pattern

```javascript
// Save to storage
async function saveSettings(settings) {
  await chrome.storage.local.set({ settings });
}

// Load from storage
async function loadSettings() {
  const result = await chrome.storage.local.get('settings');
  return result.settings || { enabled: true }; // Default
}

// Watch for changes
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'local' && changes.settings) {
    console.log('Settings changed:', changes.settings.newValue);
  }
});
```

## Icon Guidelines

| Size | Purpose | Notes |
|------|---------|-------|
| 16x16 | Toolbar | Shown in browser toolbar |
| 32x32 | Windows | Windows taskbar |
| 48x48 | Extensions page | `chrome://extensions` |
| 128x128 | Installation | Store listing, install dialog |

**Style:**
- PNG format with transparency
- Simple, recognizable at small sizes
- Use Catppuccin palette colors
- Same design, scaled appropriately

## Workflow

### Step 1: Create Project Structure

```bash
mkdir my-extension && cd my-extension
mkdir -p icons popup content background .github/assets .github/workflows
```

### Step 2: Create manifest.json

Copy the Manifest V3 template and customize:
- Update name, description, version
- Adjust permissions (minimum required)
- Remove unused sections (content_scripts if not needed, etc.)

### Step 3: Create Popup UI

Use the popup template with Catppuccin colors. Keep it minimal.

### Step 4: Add Content/Background Scripts

Only add if needed:
- Content script: Interact with web pages
- Service worker: Background tasks, cross-tab communication

### Step 5: Create Icons

Generate icon set in all required sizes. Use consistent design.

### Step 6: Add Build Automation

Use `project-ci-cd` skill for Makefile that creates distributable zip.

### Step 7: Create README

Use `project-readme` skill with Chrome Extension template. Add security disclaimer if extension handles sensitive data.

## References

| File | Purpose |
|------|---------|
| `./references/popup-template.md` | Complete popup HTML/CSS/JS with Catppuccin theme |

## Related Skills

| Skill | Use For |
|-------|---------|
| `project-readme` | README template with security disclaimer |
| `project-ci-cd` | Makefile for building extension zip |
