# Frontend Architecture - Refactored

This document describes the new modular architecture for the distributed load tester frontend.

## Project Structure

```
frontend/
├── App.jsx                     # Legacy monolithic file (can be removed after migration)
├── App_New.jsx                 # New main app entry point
├── src/
│   ├── index.js               # Central export file
│   ├── contexts/
│   │   └── AuthContext.jsx    # Authentication context and hooks
│   ├── components/
│   │   ├── common/
│   │   │   ├── UIComponents.jsx    # Reusable UI components (LoadingSpinner, StatusBadge)
│   │   │   └── Pagination.jsx      # Pagination component
│   │   ├── layout/
│   │   │   └── Navbar.jsx          # Navigation bar component
│   │   └── modals/
│   │       └── TestDetailModal.jsx # Test detail modal with tabs
│   ├── pages/
│   │   ├── LoginPage.jsx      # Login page component
│   │   └── TestHistoryPage.jsx # Test history page component
│   └── utils/
│       ├── constants.js       # Application constants and configurations
│       ├── formatters.js      # Utility functions for formatting data
│       └── api.js            # API utility functions
```

## Key Improvements

### 1. **DRY Principle Applied**
- **Shared Utilities**: Common formatting functions (`formatDate`, `formatDuration`, etc.) are centralized in `utils/formatters.js`
- **Reusable Components**: UI components like `LoadingSpinner`, `StatusBadge`, and `Pagination` are extracted and shared
- **API Functions**: All API calls are centralized in `utils/api.js`
- **Constants**: All configuration and constants are in `utils/constants.js`

### 2. **Separation of Concerns**
- **Context**: Authentication logic is isolated in `AuthContext.jsx`
- **Components**: UI components are organized by purpose (common, layout, modals)
- **Pages**: Page-level components handle routing and overall page logic
- **Utils**: Business logic and utilities are separated from UI concerns

### 3. **Improved Maintainability**
- **Single Responsibility**: Each file has a clear, single purpose
- **Easy Testing**: Components can be tested in isolation
- **Better Imports**: Central export file makes imports cleaner
- **Type Safety Ready**: Structure is ready for TypeScript migration

### 4. **Reusability**
- **Component Library**: Common components can be reused across pages
- **Utility Functions**: Formatting and API functions are reusable
- **Consistent Patterns**: Similar components follow the same patterns

## Usage Examples

### Importing Components
```jsx
// Single import for multiple components
import {
    LoadingSpinner,
    StatusBadge,
    formatDate,
    authenticatedFetch
} from './src/index.js';

// Or direct imports
import { useAuth } from './src/contexts/AuthContext.jsx';
import { TestDetailModal } from './src/components/modals/TestDetailModal.jsx';
```

### Using Utility Functions
```jsx
// Format data consistently across the app
const displayDate = formatDate(test.createdAt);
const displayDuration = formatDuration(test.durationSeconds);
const rateValue = getValueWithFallback(test, ['ratePerSecond', 'rate_per_second']);
```

### API Calls
```jsx
// Centralized API functions
try {
    const tests = await fetchTestHistory();
    const testDetail = await fetchTestDetail(testId);
    await replayTest(testId, testName);
} catch (error) {
    console.error('API Error:', error);
}
```

## Benefits

1. **Reduced Code Duplication**: Common functionality is written once and reused
2. **Easier Maintenance**: Changes to shared functionality only need to be made in one place
3. **Better Organization**: Clear separation of concerns makes the codebase easier to navigate
4. **Improved Testing**: Smaller, focused components are easier to test
5. **Enhanced Performance**: Potential for better code splitting and lazy loading
6. **Team Collaboration**: Multiple developers can work on different components without conflicts

## Migration Path

1. **Phase 1**: Create new modular structure (✅ Complete)
2. **Phase 2**: Update main App.jsx to use new components
3. **Phase 3**: Add missing pages (Dashboard, NewTest)
4. **Phase 4**: Remove legacy App.jsx file
5. **Phase 5**: Add TypeScript support (optional)

## Next Steps

1. Replace the old `App.jsx` with `App_New.jsx`
2. Implement Dashboard and NewTest pages using the same patterns
3. Add unit tests for utility functions and components
4. Consider adding PropTypes or TypeScript for better type safety
