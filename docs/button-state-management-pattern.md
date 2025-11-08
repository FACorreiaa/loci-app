# Button State Management Pattern for HTMX Forms

## Overview
This document describes the pattern for implementing proper loading state management for HTMX form submissions using Alpine.js. This ensures buttons are disabled during requests and show appropriate loading indicators.

---

## Changes Made in Discover Page

### 1. Form Container Changes

**Before:**
```templ
<form
    hx-post="/discover/search"
    hx-target="#discover-results"
    hx-indicator="#search-loading"
    hx-on::after-request="if(event.detail.successful) { htmx.ajax('GET', '/discover/recent', {target:'#recent-discoveries-section', swap:'innerHTML'}); }"
    class="relative htmx-form"
>
```

**After:**
```templ
<form
    hx-post="/discover/search"
    hx-target="#discover-results"
    hx-on::after-request="if(event.detail.successful) { htmx.ajax('GET', '/discover/recent', {target:'#recent-discoveries-section', swap:'innerHTML'}); }"
    class="relative"
    x-data="{ searching: false }"
    @htmx:before-request="searching = true"
    @htmx:after-request="searching = false"
>
```

**Key Changes:**
- ✅ Added Alpine.js state management: `x-data="{ searching: false }"`
- ✅ Added event listeners for HTMX lifecycle:
  - `@htmx:before-request="searching = true"` - Sets loading state when request starts
  - `@htmx:after-request="searching = false"` - Clears loading state when request completes
- ❌ Removed `hx-indicator="#search-loading"` - Replaced with Alpine.js state management
- ❌ Removed `htmx-form` class - No longer needed

---

### 2. Search Input Field

**Before:**
```templ
<input
    type="text"
    id="search-query"
    name="query"
    placeholder="What are you looking for? (e.g., 'best ramen in Tokyo')"
    class="w-full pl-12 pr-4 py-3 rounded-lg border focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-background text-base"
/>
```

**After:**
```templ
<input
    type="text"
    id="search-query"
    name="query"
    placeholder="What are you looking for? (e.g., 'best ramen in Tokyo')"
    class="w-full pl-12 pr-4 py-3 rounded-lg border focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-background text-base"
    x-bind:disabled="searching"
/>
```

**Key Changes:**
- ✅ Added `x-bind:disabled="searching"` - Disables input during search

---

### 3. Location Input Field

**Before:**
```templ
<input
    type="text"
    name="location"
    placeholder="Location"
    class="px-4 py-3 rounded-lg border focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-background w-32 md:w-40"
/>
```

**After:**
```templ
<input
    type="text"
    name="location"
    placeholder="Location"
    class="px-4 py-3 rounded-lg border focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-background w-32 md:w-40"
    x-bind:disabled="searching"
/>
```

**Key Changes:**
- ✅ Added `x-bind:disabled="searching"` - Disables input during search

---

### 4. Submit Button (Most Important)

**Before:**
```templ
<button
    type="submit"
    class="px-6 py-3 bg-gradient-to-r from-purple-600 to-pink-600 text-white rounded-lg
           hover:from-purple-700 hover:to-pink-700 transition-all font-medium
           flex items-center justify-center disabled:opacity-70 disabled:cursor-not-allowed
           htmx-button"
>
    <span class="search-label flex items-center gap-2">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            ></path>
        </svg>
        Search
    </span>

    <span id="search-loading" class="htmx-indicator hidden items-center">
        <div class="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin mr-2"></div>
        Searching...
    </span>
</button>
```

**After:**
```templ
<button
    type="submit"
    class="px-6 py-3 bg-gradient-to-r from-purple-600 to-pink-600 text-white rounded-lg
           hover:from-purple-700 hover:to-pink-700 transition-all font-medium
           flex items-center justify-center disabled:opacity-70 disabled:cursor-not-allowed"
    x-bind:disabled="searching"
>
    <span x-show="!searching" class="flex items-center gap-2">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            ></path>
        </svg>
        Search
    </span>
    <span x-show="searching" class="flex items-center gap-2">
        <div class="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin mr-2"></div>
        Searching...
    </span>
</button>
```

**Key Changes:**
- ✅ Added `x-bind:disabled="searching"` - Disables button during search
- ✅ Replaced conditional rendering:
  - Changed `<span class="search-label">` to `<span x-show="!searching">` - Shows when NOT searching
  - Changed `<span id="search-loading" class="htmx-indicator hidden">` to `<span x-show="searching">` - Shows ONLY when searching
- ❌ Removed `htmx-button` class - No longer needed
- ❌ Removed `search-label` class - Replaced with Alpine.js directive
- ❌ Removed `id="search-loading"` - No longer needed with Alpine.js
- ❌ Removed `htmx-indicator hidden` classes - Replaced with `x-show` directive

---

## Implementation Pattern for Other Pages

### Step 1: Add Alpine.js State to Form
```templ
<form
    hx-post="/your-endpoint"
    hx-target="#your-target"
    x-data="{ loading: false }"
    @htmx:before-request="loading = true"
    @htmx:after-request="loading = false"
>
```

**Notes:**
- Replace `loading` with a descriptive name (e.g., `searching`, `saving`, `submitting`)
- The state name should match what you use in the rest of the form

---

### Step 2: Bind Disabled State to Form Inputs
```templ
<input
    type="text"
    name="your-field"
    x-bind:disabled="loading"
/>

<textarea
    name="your-textarea"
    x-bind:disabled="loading"
></textarea>

<select
    name="your-select"
    x-bind:disabled="loading"
>
    <option>...</option>
</select>
```

**Notes:**
- Add `x-bind:disabled="loading"` to ALL form inputs
- This prevents users from changing values during submission

---

### Step 3: Update Submit Button

#### Basic Pattern
```templ
<button
    type="submit"
    class="your-classes disabled:opacity-70 disabled:cursor-not-allowed"
    x-bind:disabled="loading"
>
    <span x-show="!loading" class="flex items-center gap-2">
        <!-- Your icon SVG -->
        Button Text
    </span>
    <span x-show="loading" class="flex items-center gap-2">
        <div class="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
        Loading Text...
    </span>
</button>
```

#### With Icon Example
```templ
<button
    type="submit"
    class="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-all
           disabled:opacity-70 disabled:cursor-not-allowed"
    x-bind:disabled="saving"
>
    <span x-show="!saving" class="flex items-center gap-2">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M5 13l4 4L19 7"></path>
        </svg>
        Save Changes
    </span>
    <span x-show="saving" class="flex items-center gap-2">
        <div class="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
        Saving...
    </span>
</button>
```

---

## Complete Example: Login Form

```templ
templ LoginForm() {
    <form
        hx-post="/auth/login"
        hx-target="#login-response"
        x-data="{ loggingIn: false }"
        @htmx:before-request="loggingIn = true"
        @htmx:after-request="loggingIn = false"
        class="space-y-4"
    >
        <div>
            <label for="email" class="block text-sm font-medium mb-2">Email</label>
            <input
                type="email"
                id="email"
                name="email"
                required
                class="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500"
                x-bind:disabled="loggingIn"
            />
        </div>

        <div>
            <label for="password" class="block text-sm font-medium mb-2">Password</label>
            <input
                type="password"
                id="password"
                name="password"
                required
                class="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500"
                x-bind:disabled="loggingIn"
            />
        </div>

        <button
            type="submit"
            class="w-full px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700
                   transition-all disabled:opacity-70 disabled:cursor-not-allowed"
            x-bind:disabled="loggingIn"
        >
            <span x-show="!loggingIn" class="flex items-center justify-center gap-2">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1"></path>
                </svg>
                Log In
            </span>
            <span x-show="loggingIn" class="flex items-center justify-center gap-2">
                <div class="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                Logging In...
            </span>
        </button>

        <div id="login-response"></div>
    </form>
}
```

---

## Complete Example: Contact Form

```templ
templ ContactForm() {
    <form
        hx-post="/contact/submit"
        hx-target="#contact-response"
        x-data="{ submitting: false }"
        @htmx:before-request="submitting = true"
        @htmx:after-request="submitting = false"
        class="space-y-4"
    >
        <div>
            <label for="name" class="block text-sm font-medium mb-2">Name</label>
            <input
                type="text"
                id="name"
                name="name"
                required
                class="w-full px-4 py-2 border rounded-lg"
                x-bind:disabled="submitting"
            />
        </div>

        <div>
            <label for="email" class="block text-sm font-medium mb-2">Email</label>
            <input
                type="email"
                id="email"
                name="email"
                required
                class="w-full px-4 py-2 border rounded-lg"
                x-bind:disabled="submitting"
            />
        </div>

        <div>
            <label for="message" class="block text-sm font-medium mb-2">Message</label>
            <textarea
                id="message"
                name="message"
                rows="4"
                required
                class="w-full px-4 py-2 border rounded-lg"
                x-bind:disabled="submitting"
            ></textarea>
        </div>

        <button
            type="submit"
            class="px-6 py-3 bg-green-600 text-white rounded-lg hover:bg-green-700
                   transition-all disabled:opacity-70 disabled:cursor-not-allowed"
            x-bind:disabled="submitting"
        >
            <span x-show="!submitting" class="flex items-center gap-2">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                          d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
                </svg>
                Send Message
            </span>
            <span x-show="submitting" class="flex items-center gap-2">
                <div class="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                Sending...
            </span>
        </button>

        <div id="contact-response"></div>
    </form>
}
```

---

## Key Benefits

1. ✅ **User Feedback**: Clear visual indication when a request is in progress
2. ✅ **Prevent Double Submissions**: Disabled state prevents multiple form submissions
3. ✅ **Better UX**: Users can't interact with the form while it's processing
4. ✅ **Clean State Management**: Alpine.js provides reactive state without extra JavaScript
5. ✅ **Accessible**: Disabled state is properly communicated to screen readers
6. ✅ **Consistent**: Same pattern across all forms in the application

---

## Required Dependencies

### 1. Alpine.js
Must be included in your HTML layout:
```html
<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
```

### 2. HTMX
For handling AJAX requests:
```html
<script src="https://unpkg.com/htmx.org@1.9.x"></script>
```

### 3. Tailwind CSS
For styling (especially `disabled:` variants):
```html
<link href="https://cdn.jsdelivr.net/npm/tailwindcss@3.x.x/dist/tailwind.min.css" rel="stylesheet">
```

---

## Testing Checklist

When implementing on other pages, verify:

- [ ] Button shows loading state when clicked
- [ ] Button is disabled during request
- [ ] All form inputs are disabled during request
- [ ] Button returns to normal state after request completes (success)
- [ ] Button returns to normal state after request fails (error)
- [ ] Multiple rapid clicks don't trigger multiple requests
- [ ] Loading spinner animates properly
- [ ] Text changes from action text to loading text
- [ ] Screen readers announce disabled state
- [ ] Tab navigation skips disabled elements
- [ ] Form can be submitted again after request completes

---

## Common State Names

Use descriptive state names that match the action:

| Action | State Variable Name | Button Text |
|--------|-------------------|-------------|
| Search | `searching` | "Search" → "Searching..." |
| Save | `saving` | "Save" → "Saving..." |
| Submit | `submitting` | "Submit" → "Submitting..." |
| Login | `loggingIn` | "Log In" → "Logging In..." |
| Send | `sending` | "Send" → "Sending..." |
| Create | `creating` | "Create" → "Creating..." |
| Update | `updating` | "Update" → "Updating..." |
| Delete | `deleting` | "Delete" → "Deleting..." |
| Upload | `uploading` | "Upload" → "Uploading..." |
| Download | `downloading` | "Download" → "Downloading..." |

---

## Troubleshooting

### Button doesn't disable
- ✅ Check Alpine.js is loaded
- ✅ Verify `x-data` is on the form element
- ✅ Ensure `x-bind:disabled` matches the state variable name

### Loading state doesn't show
- ✅ Check `x-show` directives are correct (`!loading` vs `loading`)
- ✅ Verify event listeners are set: `@htmx:before-request` and `@htmx:after-request`

### Multiple requests still happen
- ✅ Ensure `x-bind:disabled` is on the button
- ✅ Check that the state is being set to `true` before the request

### Button stays disabled after error
- ✅ Make sure `@htmx:after-request` event fires (it should fire on both success and error)
- ✅ Check browser console for JavaScript errors

---

## Pages to Update

List of pages that need this pattern applied:

- [ ] `/tags` - Tag management forms
- [ ] `/chat_prompt` - Chat/prompt submission
- [ ] `/profiles` - Profile edit forms
- [ ] `/interests/itinerary` - Itinerary creation/edit forms
- [ ] Any other HTMX forms in the application

---

## Notes

- This pattern replaces the old HTMX `hx-indicator` approach
- Alpine.js provides better control and flexibility
- The pattern is framework-agnostic and can work with any HTMX setup
- Consider creating a reusable templ component for the button loading states

---

**Last Updated:** 2025-11-08
**Version:** 1.0
**Author:** Documentation based on discover page implementation
