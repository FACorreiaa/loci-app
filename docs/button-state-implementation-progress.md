# Button State Management Implementation Progress

## Overview
This document tracks the implementation of the Alpine.js + HTMX button state management pattern across all forms in the application.

---

## ✅ Completed Implementations

### 1. Authentication Forms (Priority 1 - COMPLETE)

#### ✅ Sign In Form (`auth/signin.templ`)
- **Status:** COMPLETED
- **Changes:**
  - Added Alpine.js state: `x-data="{ signingIn: false }"`
  - Added HTMX event listeners: `@htmx:before-request` and `@htmx:after-request`
  - Disabled email and password inputs during submission
  - Updated button with conditional rendering (normal state / loading state)
- **State Variable:** `signingIn`
- **Button Text:** "Sign in" → "Signing in..."

#### ✅ Sign Up Form (`auth/signup.templ`)
- **Status:** COMPLETED
- **Changes:**
  - Added Alpine.js state: `x-data="{ signingUp: false }"`
  - Added HTMX event listeners
  - Disabled all inputs during submission:
    - First name
    - Last name
    - Email
    - Password
    - Confirm password
    - Terms checkbox
  - Updated button with conditional rendering
- **State Variable:** `signingUp`
- **Button Text:** "Create account" → "Creating account..."

#### ✅ Forgot Password Form (`auth/forgot-password.templ`)
- **Status:** COMPLETED
- **Changes:**
  - Added Alpine.js state: `x-data="{ sending: false }"`
  - Added HTMX event listeners
  - Disabled email input during submission
  - Updated button with conditional rendering
- **State Variable:** `sending`
- **Button Text:** "Send reset link" → "Sending reset link..."

---

## 🔄 Pending Implementations

### 2. Search Forms (Priority 2 - HIGH)

#### ⏳ Landing Page Search (`pages/landing.templ`)
- **Location:** Lines 23-46
- **Current State:** Uses `hx-indicator="#loading-spinner"` (old pattern)
- **Required Changes:**
  1. Add Alpine.js state to container: `x-data="{ searching: false }"`
  2. Add event listeners: `@htmx:before-request="searching = true"` and `@htmx:after-request="searching = false"`
  3. Disable textarea during search: `x-bind:disabled="searching"`
  4. Update button:
     ```templ
     <button
         id="search-btn"
         hx-post="/search"
         hx-include="#search-input"
         hx-target="#search-results"
         class="px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white rounded-xl font-medium transition-colors flex items-center gap-2"
         x-bind:disabled="searching"
     >
         <span x-show="!searching" class="flex items-center gap-2">
             <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                 <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"></path>
             </svg>
             <span class="hidden sm:inline">Try Free</span>
         </span>
         <span x-show="searching" class="flex items-center gap-2">
             <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
             <span class="hidden sm:inline">Searching...</span>
         </span>
     </button>
     ```
  5. Remove `hx-indicator` and `#loading-spinner` div (no longer needed)

- **State Variable:** `searching`
- **Button Text:** "Try Free" → "Searching..."

#### ⏳ Discover Page is already implemented ✅
- **Status:** ALREADY COMPLETED (reference implementation)
- **File:** `discover/discover.templ`

### 3. Chat Forms (Priority 2)

#### ⏳ Chat Message Form (`chat_prompt/chat.templ`)
- **Location:** Lines 93-100
- **Current State:** Uses inline onclick handler
- **Required Changes:**
  1. Wrap in form with Alpine.js state
  2. Add disabled state to textarea
  3. Update send button similar to search forms
- **State Variable:** `sending`
- **Button Text:** "Send" → "Sending..."

### 4. Profile Forms (Priority 3)

#### ⏳ Edit Profile Form (`profiles/profile.templ`)
- **Location:** Line 75
- **Current State:** No loading indicator
- **Required Changes:**
  1. Add Alpine.js state: `x-data="{ saving: false }"`
  2. Add HTMX event listeners
  3. Disable all profile input fields
  4. Update submit button
- **State Variable:** `saving`
- **Button Text:** "Save Changes" → "Saving..."

#### ⏳ Settings Forms (`settings/settings.templ`)
Multiple forms need implementation:
1. **Profile Update:** `x-data="{ updating: false }"`
2. **Password Change:** `x-data="{ changing: false }"`
3. **Delete Account:** `x-data="{ deleting: false }"`

### 5. Other Forms (Priority 4)

#### ⏳ Navbar Logout (`components/navbar/navbar.templ`)
- **Type:** Single button action
- **State Variable:** `loggingOut`
- **Button Text:** "Logout" → "Logging out..."

#### ⏳ List Management (`lists/lists.templ`)
- Multiple CRUD operations need implementation

---

## Implementation Template

### For Standard Forms

```templ
<form
    hx-post="/your-endpoint"
    hx-target="#response-target"
    x-data="{ [stateName]: false }"
    @htmx:before-request="[stateName] = true"
    @htmx:after-request="[stateName] = false"
>
    <input
        type="text"
        name="field"
        x-bind:disabled="[stateName]"
        class="... disabled:opacity-50 disabled:cursor-not-allowed"
    />

    <button
        type="submit"
        class="... disabled:opacity-50 disabled:cursor-not-allowed"
        x-bind:disabled="[stateName]"
    >
        <span x-show="![stateName]" class="flex items-center gap-2">
            Button Text
        </span>
        <span x-show="[stateName]" class="flex items-center gap-2">
            <div class="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
            Loading Text...
        </span>
    </button>
</form>
```

### For Single Action Buttons (No Form)

```templ
<div
    x-data="{ [stateName]: false }"
    @htmx:before-request="[stateName] = true"
    @htmx:after-request="[stateName] = false"
>
    <button
        hx-post="/your-endpoint"
        class="... disabled:opacity-50 disabled:cursor-not-allowed"
        x-bind:disabled="[stateName]"
    >
        <span x-show="![stateName]">Action</span>
        <span x-show="[stateName]" class="flex items-center gap-2">
            <div class="w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin"></div>
            Acting...
        </span>
    </button>
</div>
```

---

## State Variable Naming Convention

| Action | State Variable | Button States |
|--------|---------------|---------------|
| Sign In | `signingIn` | "Sign in" → "Signing in..." |
| Sign Up | `signingUp` | "Create account" → "Creating account..." |
| Send | `sending` | "Send" → "Sending..." |
| Search | `searching` | "Search" → "Searching..." |
| Save | `saving` | "Save" → "Saving..." |
| Update | `updating` | "Update" → "Updating..." |
| Delete | `deleting` | "Delete" → "Deleting..." |
| Create | `creating` | "Create" → "Creating..." |
| Submit | `submitting` | "Submit" → "Submitting..." |
| Log Out | `loggingOut` | "Log out" → "Logging out..." |

---

## Files Modified

### ✅ Completed
1. ✅ `/internal/app/domain/auth/signin.templ`
2. ✅ `/internal/app/domain/auth/signup.templ`
3. ✅ `/internal/app/domain/auth/forgot-password.templ`
4. ✅ `/internal/app/domain/discover/discover.templ` (already done)

### ⏳ Remaining
5. ⏳ `/internal/app/domain/pages/landing.templ`
6. ⏳ `/internal/app/domain/chat_prompt/chat.templ`
7. ⏳ `/internal/app/domain/profiles/profile.templ`
8. ⏳ `/internal/app/domain/settings/settings.templ`
9. ⏳ `/internal/app/components/navbar/navbar.templ`
10. ⏳ `/internal/app/domain/lists/lists.templ`
11. ⏳ `/internal/app/domain/activities/activities.templ`
12. ⏳ `/internal/app/domain/restaurants/restaurants.templ`
13. ⏳ `/internal/app/domain/hotels/hotels.templ`

---

## Testing Checklist

After implementing each form, verify:

- [ ] Button shows loading state when clicked
- [ ] Button is disabled during request
- [ ] All form inputs are disabled during request
- [ ] Button returns to normal state after success
- [ ] Button returns to normal state after error
- [ ] Multiple rapid clicks don't trigger multiple requests
- [ ] Loading spinner animates smoothly
- [ ] Text changes correctly (action → loading)
- [ ] Focus returns to form after completion (if applicable)

---

## Notes

- **Alpine.js Requirement:** Ensure Alpine.js is loaded in the base layout template
- **Consistency:** Use the exact same pattern across all forms for maintainability
- **Accessibility:** The `disabled` attribute is properly announced by screen readers
- **Performance:** Alpine.js adds minimal overhead and provides reactive state management

---

**Last Updated:** 2025-11-08
**Status:** 4/18+ forms completed (22% complete)
**Next Priority:** Landing page search form
