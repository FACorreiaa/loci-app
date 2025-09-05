const CACHE_NAME = 'go-templui-v1';
const urlsToCache = [
    '/',
    '/assets/css/output.css',
    '/assets/static/manifest.json',
    '/assets/static/offline.html'
];

// Install the service worker and cache assets
self.addEventListener('install', event => {
    event.waitUntil(
        caches.open(CACHE_NAME).then(cache => {
            return cache.addAll(urlsToCache);
        })
    );
    self.skipWaiting();
});

// Activate the service worker and clean up old caches
self.addEventListener('activate', event => {
    event.waitUntil(
        caches.keys().then(cacheNames => {
            return Promise.all(
                cacheNames
                    .filter(name => name !== CACHE_NAME)
                    .map(name => caches.delete(name))
            );
        })
    );
    self.clients.claim();
});

// Fetch event: Serve cached content or fetch from network
self.addEventListener('fetch', event => {
    event.respondWith(
        caches.match(event.request).then(response => {
            // Return cached response if available, otherwise fetch from network
            return response || fetch(event.request).catch(() => {
                // For navigation requests, show offline page
                if (event.request.mode === 'navigate') {
                    return caches.match('/assets/static/offline.html');
                }
                // For other requests, try to return the homepage or offline page
                return caches.match('/') || caches.match('/assets/static/offline.html');
            });
        })
    );
});