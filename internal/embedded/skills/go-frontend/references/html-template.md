# HTML Template

## Full Template (Dark Theme Only)

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>APP_NAME</title>

    <!-- Icons -->
    <link rel="icon" type="image/x-icon" href="/static/icons/favicon.ico">
    <link rel="icon" type="image/png" sizes="32x32" href="/static/icons/favicon.png">
    <link rel="apple-touch-icon" sizes="180x180" href="/static/icons/apple-touch-icon.png">

    <!-- PWA (remove if not needed) -->
    <link rel="manifest" href="/static/manifest.json">
    <meta name="theme-color" content="#1e1e2e">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
    <meta name="apple-mobile-web-app-title" content="APP_NAME">

    <!-- Stylesheets -->
    <link rel="stylesheet" href="/static/css/inter.css">
    <link rel="stylesheet" href="/static/css/jetbrains-mono.css">

    <!-- Icon Libraries (use what you need) -->
    <!-- Lucide (preferred) - initialize with lucide.createIcons() -->
    <script src="/static/js/lucide.min.js"></script>
    <!-- Font Awesome (fallback, brand icons) -->
    <link rel="stylesheet" href="/static/fontawesome/css/all.min.css">
    <!-- Dev Icons (tech logos) -->
    <link rel="stylesheet" href="/static/css/devicon.min.css">

    <!-- Catppuccin Mocha (Dark Theme) -->
    <style>
        :root {
            --rosewater: #f5e0dc; --flamingo: #f2cdcd; --pink: #f5c2e7;
            --mauve: #cba6f7; --red: #f38ba8; --maroon: #eba0ac;
            --peach: #fab387; --yellow: #f9e2af; --green: #a6e3a1;
            --teal: #94e2d5; --sky: #89dceb; --sapphire: #74c7ec;
            --blue: #89b4fa; --lavender: #b4befe; --text: #cdd6f4;
            --subtext1: #bac2de; --subtext0: #a6adc8; --overlay2: #9399b2;
            --overlay1: #7f849c; --overlay0: #6c7086; --surface2: #585b70;
            --surface1: #45475a; --surface0: #313244; --base: #1e1e2e;
            --mantle: #181825; --crust: #11111b;
        }
        body {
            font-family: 'Inter', sans-serif;
        }
    </style>

    <script src="/static/js/tailwindcss.js"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    borderRadius: {
                        '4xl': '2rem',
                    },
                    colors: {
                        'rosewater': 'var(--rosewater)', 'flamingo': 'var(--flamingo)',
                        'pink': 'var(--pink)', 'mauve': 'var(--mauve)',
                        'red': 'var(--red)', 'maroon': 'var(--maroon)',
                        'peach': 'var(--peach)', 'yellow': 'var(--yellow)',
                        'green': 'var(--green)', 'teal': 'var(--teal)',
                        'sky': 'var(--sky)', 'sapphire': 'var(--sapphire)',
                        'blue': 'var(--blue)', 'lavender': 'var(--lavender)',
                        'text': 'var(--text)', 'subtext1': 'var(--subtext1)',
                        'subtext0': 'var(--subtext0)', 'overlay2': 'var(--overlay2)',
                        'overlay1': 'var(--overlay1)', 'overlay0': 'var(--overlay0)',
                        'surface2': 'var(--surface2)', 'surface1': 'var(--surface1)',
                        'surface0': 'var(--surface0)', 'base': 'var(--base)',
                        'mantle': 'var(--mantle)', 'crust': 'var(--crust)',
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-base text-text min-h-screen">
    <!-- Navigation (if needed) -->
    <nav class="bg-mantle border-b border-surface0 px-4 py-3">
        <div class="max-w-6xl mx-auto flex items-center justify-between">
            <div class="flex items-center gap-2">
                <img src="/static/icons/logo.png" alt="Logo" class="w-8 h-8">
                <span class="text-lg font-semibold">APP_NAME</span>
            </div>
            <div class="flex items-center gap-4">
                <a href="#" class="text-subtext1 hover:text-text transition-colors">Link</a>
            </div>
        </div>
    </nav>

    <!-- Main Content -->
    <main class="max-w-6xl mx-auto px-4 py-8">
        <h1 class="text-2xl font-bold mb-6">Page Title</h1>

        <!-- Content here -->

    </main>

    <!-- Toast Container -->
    <div id="toast" class="fixed bottom-4 right-4 hidden z-50">
        <div class="bg-surface0 text-text px-4 py-2 rounded-lg shadow-lg border border-surface1">
            <span id="toast-message"></span>
        </div>
    </div>

    <!-- Application JavaScript -->
    <script src="/static/app.js"></script>

    <!-- Initialize Lucide icons -->
    <script>lucide.createIcons();</script>

    <!-- Service Worker Registration (remove if no PWA) -->
    <script>
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/static/sw.js');
        }
    </script>
</body>
</html>
```

## Full Template (With Light/Dark Theme Switching)

Only use when explicitly requested.

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>APP_NAME</title>

    <!-- Icons -->
    <link rel="icon" type="image/x-icon" href="/static/icons/favicon.ico">
    <link rel="icon" type="image/png" sizes="32x32" href="/static/icons/favicon.png">
    <link rel="apple-touch-icon" sizes="180x180" href="/static/icons/apple-touch-icon.png">

    <!-- PWA (remove if not needed) -->
    <link rel="manifest" href="/static/manifest.json">
    <meta name="theme-color" content="#1e1e2e">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
    <meta name="apple-mobile-web-app-title" content="APP_NAME">

    <!-- Stylesheets -->
    <link rel="stylesheet" href="/static/css/inter.css">
    <link rel="stylesheet" href="/static/css/jetbrains-mono.css">

    <!-- Icon Libraries (use what you need) -->
    <!-- Lucide (preferred) - initialize with lucide.createIcons() -->
    <script src="/static/js/lucide.min.js"></script>
    <!-- Font Awesome (fallback, brand icons) -->
    <link rel="stylesheet" href="/static/fontawesome/css/all.min.css">
    <!-- Dev Icons (tech logos) -->
    <link rel="stylesheet" href="/static/css/devicon.min.css">

    <!-- Catppuccin Theme Variables -->
    <style>
        :root { /* Catppuccin Latte (Light Theme) */
            --rosewater: #dc8a78; --flamingo: #dd7878; --pink: #ea76cb;
            --mauve: #8839ef; --red: #d20f39; --maroon: #e64553;
            --peach: #fe640b; --yellow: #df8e1d; --green: #40a02b;
            --teal: #179299; --sky: #04a5e5; --sapphire: #209fb5;
            --blue: #1e66f5; --lavender: #7287fd; --text: #4c4f69;
            --subtext1: #5c5f77; --subtext0: #6c6f85; --overlay2: #7c7f93;
            --overlay1: #8c8fa1; --overlay0: #9ca0b0; --surface2: #acb0be;
            --surface1: #bcc0cc; --surface0: #ccd0da; --base: #eff1f5;
            --mantle: #e6e9ef; --crust: #dce0e8;
        }

        html.dark { /* Catppuccin Mocha (Dark Theme) */
            --rosewater: #f5e0dc; --flamingo: #f2cdcd; --pink: #f5c2e7;
            --mauve: #cba6f7; --red: #f38ba8; --maroon: #eba0ac;
            --peach: #fab387; --yellow: #f9e2af; --green: #a6e3a1;
            --teal: #94e2d5; --sky: #89dceb; --sapphire: #74c7ec;
            --blue: #89b4fa; --lavender: #b4befe; --text: #cdd6f4;
            --subtext1: #bac2de; --subtext0: #a6adc8; --overlay2: #9399b2;
            --overlay1: #7f849c; --overlay0: #6c7086; --surface2: #585b70;
            --surface1: #45475a; --surface0: #313244; --base: #1e1e2e;
            --mantle: #181825; --crust: #11111b;
        }

        body {
            font-family: 'Inter', sans-serif;
        }
    </style>

    <!-- Theme Detection (Runs Immediately to Prevent FOUC) -->
    <script>
        (function() {
            function applyTheme(theme) {
                if (theme === 'dark') {
                    document.documentElement.classList.add('dark');
                } else {
                    document.documentElement.classList.remove('dark');
                }
            }

            // Check for saved preference, otherwise use system preference
            const savedTheme = localStorage.getItem('theme');
            if (savedTheme) {
                applyTheme(savedTheme);
            } else {
                const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
                applyTheme(mediaQuery.matches ? 'dark' : 'light');

                // Listen for system theme changes
                mediaQuery.addEventListener('change', (e) => {
                    if (!localStorage.getItem('theme')) {
                        applyTheme(e.matches ? 'dark' : 'light');
                    }
                });
            }
        })();
    </script>

    <script src="/static/js/tailwindcss.js"></script>
    <script>
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    borderRadius: {
                        '4xl': '2rem',
                    },
                    colors: {
                        'rosewater': 'var(--rosewater)', 'flamingo': 'var(--flamingo)',
                        'pink': 'var(--pink)', 'mauve': 'var(--mauve)',
                        'red': 'var(--red)', 'maroon': 'var(--maroon)',
                        'peach': 'var(--peach)', 'yellow': 'var(--yellow)',
                        'green': 'var(--green)', 'teal': 'var(--teal)',
                        'sky': 'var(--sky)', 'sapphire': 'var(--sapphire)',
                        'blue': 'var(--blue)', 'lavender': 'var(--lavender)',
                        'text': 'var(--text)', 'subtext1': 'var(--subtext1)',
                        'subtext0': 'var(--subtext0)', 'overlay2': 'var(--overlay2)',
                        'overlay1': 'var(--overlay1)', 'overlay0': 'var(--overlay0)',
                        'surface2': 'var(--surface2)', 'surface1': 'var(--surface1)',
                        'surface0': 'var(--surface0)', 'base': 'var(--base)',
                        'mantle': 'var(--mantle)', 'crust': 'var(--crust)',
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-base text-text min-h-screen">
    <!-- Content -->

    <!-- Theme Toggle Button -->
    <button id="theme-toggle" class="fixed bottom-4 left-4 p-2 rounded-lg bg-surface0 hover:bg-surface1 transition-colors">
        <i id="theme-icon" class="fas fa-moon text-text"></i>
    </button>

    <script>
        const themeToggle = document.getElementById('theme-toggle');
        const themeIcon = document.getElementById('theme-icon');

        function updateIcon() {
            const isDark = document.documentElement.classList.contains('dark');
            themeIcon.className = isDark ? 'fas fa-sun text-text' : 'fas fa-moon text-text';
        }

        themeToggle.addEventListener('click', () => {
            const isDark = document.documentElement.classList.contains('dark');
            const newTheme = isDark ? 'light' : 'dark';
            localStorage.setItem('theme', newTheme);
            document.documentElement.classList.toggle('dark');
            updateIcon();
        });

        updateIcon();
    </script>

    <!-- Initialize Lucide icons -->
    <script>lucide.createIcons();</script>

    <!-- Service Worker Registration (remove if no PWA) -->
    <script>
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/static/sw.js');
        }
    </script>
</body>
</html>
```

## PWA Files

### manifest.json

```json
{
  "name": "APP_NAME",
  "short_name": "APP_SHORT",
  "description": "APP_DESCRIPTION",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#1e1e2e",
  "theme_color": "#1e1e2e",
  "icons": [
    {
      "src": "/static/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/static/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
```

### sw.js (Service Worker)

No-op service worker - exists only to enable PWA installation. No caching, all requests go directly to network (behaves like a normal browser tab).

```javascript
// No-op service worker for PWA registration only
// All requests pass through to network (no offline caching)
self.addEventListener('fetch', () => {});
```

## Common UI Components

### Card

```html
<div class="bg-surface0 rounded-lg border border-surface1 p-4">
    <h3 class="text-lg font-semibold mb-2">Card Title</h3>
    <p class="text-subtext1">Card content goes here.</p>
</div>
```

### Button Variants

```html
<!-- Primary -->
<button class="bg-blue hover:bg-sapphire text-crust font-medium px-4 py-2 rounded-lg transition-colors">
    Primary
</button>

<!-- Secondary -->
<button class="bg-surface1 hover:bg-surface2 text-text font-medium px-4 py-2 rounded-lg transition-colors">
    Secondary
</button>

<!-- Danger -->
<button class="bg-red hover:bg-maroon text-crust font-medium px-4 py-2 rounded-lg transition-colors">
    Delete
</button>

<!-- Ghost -->
<button class="text-subtext1 hover:text-text hover:bg-surface0 font-medium px-4 py-2 rounded-lg transition-colors">
    Ghost
</button>
```

### Input Field

```html
<input type="text"
    placeholder="Enter value..."
    class="w-full bg-surface0 border border-surface1 rounded-lg px-3 py-2 text-text placeholder-overlay1 focus:outline-none focus:border-blue">
```

### Select Dropdown

```html
<select class="bg-surface0 border border-surface1 rounded-lg px-3 py-2 text-text focus:outline-none focus:border-blue">
    <option value="">Select option...</option>
    <option value="1">Option 1</option>
    <option value="2">Option 2</option>
</select>
```

### Table

```html
<div class="overflow-x-auto">
    <table class="w-full">
        <thead>
            <tr class="border-b border-surface1">
                <th class="text-left py-2 px-4 text-subtext1 font-medium">Name</th>
                <th class="text-left py-2 px-4 text-subtext1 font-medium">Status</th>
                <th class="text-left py-2 px-4 text-subtext1 font-medium">Actions</th>
            </tr>
        </thead>
        <tbody>
            <tr class="border-b border-surface0 hover:bg-surface0/50">
                <td class="py-2 px-4">Item Name</td>
                <td class="py-2 px-4"><span class="text-green">Active</span></td>
                <td class="py-2 px-4">
                    <button class="text-blue hover:text-sapphire">Edit</button>
                </td>
            </tr>
        </tbody>
    </table>
</div>
```

### Modal

```html
<div id="modal" class="fixed inset-0 bg-crust/80 hidden items-center justify-center z-50">
    <div class="bg-base rounded-lg border border-surface1 p-6 max-w-md w-full mx-4">
        <h2 class="text-xl font-semibold mb-4">Modal Title</h2>
        <p class="text-subtext1 mb-6">Modal content goes here.</p>
        <div class="flex justify-end gap-2">
            <button onclick="closeModal()" class="bg-surface1 hover:bg-surface2 text-text px-4 py-2 rounded-lg">
                Cancel
            </button>
            <button class="bg-blue hover:bg-sapphire text-crust px-4 py-2 rounded-lg">
                Confirm
            </button>
        </div>
    </div>
</div>

<script>
function openModal() {
    document.getElementById('modal').classList.remove('hidden');
    document.getElementById('modal').classList.add('flex');
}
function closeModal() {
    document.getElementById('modal').classList.add('hidden');
    document.getElementById('modal').classList.remove('flex');
}
</script>
```
