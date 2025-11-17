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

#### ✅ Landing Page Search (`pages/landing.templ`)
- **Status:** COMPLETED (was already implemented)
- **Location:** Lines 23-68
- **Implementation Details:**
  - Added Alpine.js state: `x-data="{ searching: false }"`
  - Added HTMX event listeners: `@htmx:before-request="searching = true"` and `@htmx:after-request="searching = false"`
  - Disabled textarea during search: `x-bind:disabled="searching"`
  - Updated button with conditional rendering (normal state / loading state)
  - Includes loading indicator shown during search
- **State Variable:** `searching`
- **Button Text:** "Try Free" → "Searching..."

#### ✅ Discover Page
- **Status:** COMPLETED (reference implementation)
- **File:** `discover/discover.templ`

### 3. Chat Forms (Priority 2)

#### ✅ Chat Message Form (`chat_prompt/chat.templ`)
- **Status:** COMPLETED (was already implemented)
- **Location:** Lines 88-128
- **Implementation Details:**
  - Added Alpine.js state: `x-data="{ sending: false }"`
  - Added HTMX event listeners: `@htmx:before-request="sending = true"` and `@htmx:after-request="sending = false"`
  - Disabled textarea during sending: `x-bind:disabled="sending"`
  - Updated button with loading spinner
- **State Variable:** `sending`
- **Button Text:** Icon only → Loading spinner

### 4. Profile Forms (Priority 3)

#### ✅ Edit Profile Form (`profiles/profile.templ`)
- **Status:** COMPLETED
- **Location:** Lines 75-227
- **Implementation Details:**
  - Added Alpine.js state: `x-data="{ saving: false }"`
  - Added HTMX event listeners: `@htmx:before-request="saving = true"` and `@htmx:after-request="saving = false"`
  - Disabled all profile input fields during saving:
    - First name, Last name, Email, Location, Phone, Bio
  - Updated submit button with loading state
- **State Variable:** `saving`
- **Button Text:** "Save Changes" → "Saving..."

#### ✅ Settings Forms (`settings/settings.templ`)
- **Status:** COMPLETED
- **Implementation Details:**
  1. **Profile Update Form** (lines 167-272)
     - State Variable: `updating`
     - Button Text: "Save Changes" → "Updating..."
     - Disabled inputs: First Name, Last Name, Email, City, Country, Bio
  2. **Password Change Form** (lines 1073-1119)
     - State Variable: `changing`
     - Button Text: "Update Password" → "Changing..."
     - Disabled inputs: Current Password, New Password, Confirm Password
  3. **Delete Account Button** (lines 1174-1196)
     - State Variable: `deleting`
     - Button Text: "Delete" → "Deleting..."
  4. **Verify Phone Button** (lines 1228-1255)
     - State Variable: `verifying`
     - Button Text: "Verify" → "Verifying..."
     - Disabled input: Phone number field

### 5. Other Forms (Priority 4)

#### ✅ Navbar Logout (`components/navbar/navbar.templ`)
- **Status:** COMPLETED
- **Type:** Single button action
- **State Variable:** `loggingOut`
- **Button Text:** "Sign Out" → "Signing out..."
- **Implementation Details:**
  - Added Alpine.js state to both desktop and mobile logout buttons
  - Wrapped buttons with `x-data="{ loggingOut: false }"`
  - Added HTMX event listeners: `@htmx:before-request` and `@htmx:after-request`
  - Updated button with loading spinner during logout process
  - Added disabled state with opacity and cursor styling

#### 🔄 List Management (`lists/lists.templ`)
- **Status:** IN PROGRESS (1/8 actions completed)
- **Completed:**
  1. **Create List Form** (lines 166-277)
     - State Variable: `creating`
     - Button Text: "Create List" → "Creating..."
     - Disabled inputs: Name, Description
- **Remaining:**
  - Add item to list (line 333)
  - Delete item (line 595)
  - Update list (line 816)
  - Delete list (line 948)
  - Unsave list (lines 1034, 1081)
  - Save list (line 1092)

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
4. ✅ `/internal/app/domain/discover/discover.templ`
5. ✅ `/internal/app/domain/pages/landing.templ`
6. ✅ `/internal/app/domain/chat_prompt/chat.templ`
7. ✅ `/internal/app/domain/profiles/profile.templ`
8. ✅ `/internal/app/components/navbar/navbar.templ`
9. ✅ `/internal/app/domain/settings/settings.templ`

### ⏳ Remaining
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
**Status:** 9/13 forms completed (69% complete)
**Next Priority:** Lists management, Activities/Restaurants/Hotels pages
