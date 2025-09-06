const CACHE_NAME = 'loci-pwa-v2';
const urlsToCache = [
    '/static/offline.html',
    '/assets/css/output.css',
    '/static/manifest.json',
    '/static/images/icons/icon-144x144.png',
    '/static/images/icons/icon-192x192.png',
    '/static/images/icons/icon-512x512.png',
];

self.addEventListener('install', event => {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then(cache => {
                console.log('Opened cache');
                return cache.addAll(urlsToCache);
            })
    );
    self.skipWaiting();
});

self.addEventListener('activate', event => {
    event.waitUntil(
        caches.keys().then(cacheNames => {
            return Promise.all(
                cacheNames.filter(name => name !== CACHE_NAME).map(name => caches.delete(name))
            );
        })
    );
    self.clients.claim();
});

self.addEventListener('fetch', event => {
    const { request } = event;

    // For navigation requests, use a network-first strategy
    if (request.mode === 'navigate') {
        event.respondWith(
            fetch(request)
                .then(response => {
                    // If the request is successful, cache the response
                    if (response.ok) {
                        const cacheCopy = response.clone();
                        caches.open(CACHE_NAME)
                            .then(cache => {
                                cache.put(request, cacheCopy);
                            });
                    }
                    return response;
                })
                .catch(() => {
                    // If the network fails, serve the offline page from the cache
                    return caches.match('/static/offline.html');
                })
        );
        return;
    }

    // For other requests (e.g., CSS, images), use a cache-first strategy
    event.respondWith(
        caches.match(request)
            .then(response => {
                // If the resource is in the cache, serve it
                if (response) {
                    return response;
                }

                // If the resource is not in the cache, fetch it from the network
                return fetch(request)
                    .then(networkResponse => {
                        // If the request is successful, cache the response
                        if (networkResponse.ok) {
                            const cacheCopy = networkResponse.clone();
                            caches.open(CACHE_NAME)
                                .then(cache => {
                                    cache.put(request, cacheCopy);
                                });
                        }
                        return networkResponse;
                    });
            })
    );
});
