# Frontend Architecture - Hybrid Approach Implementation

This document describes the modular hybrid architecture for the distributed load tester frontend, featuring both quick-view modals and dedicated detail pages.

## Project Structure

```
frontend/
├── App.jsx                        # Main app entry point with React Router
├── index.html                     # HTML entry point
├── package.json                   # Dependencies (includes react-router-dom)
├── vite.config.js                # Vite configuration
└── src/
    ├── index.js                   # Central export file
    ├── contexts/
    │   └── AuthContext.jsx        # Authentication context and hooks
    ├── components/
    │   ├── common/
    │   │   ├── UIComponents.jsx   # Reusable UI components (LoadingSpinner, StatusBadge)
    │   │   └── Pagination.jsx     # Pagination component
    │   ├── layout/
    │   │   └── Navbar.jsx         # Navigation bar with React Router Links
    │   └── modals/
    │       └── TestDetailModal.jsx # Quick overview modal with "View Full Details" button
    ├── pages/
    │   ├── LoginPage.jsx          # Login page component
    │   ├── DashboardPage.jsx      # Dashboard overview page
    │   ├── NewTestPage.jsx        # Create new test page
    │   ├── TestHistoryPage.jsx    # Test history listing page
    │   └── TestDetailPage.jsx     # Dedicated full test details page with charts
    └── utils/
        ├── constants.js           # Application constants and configurations
        ├── formatters.js          # Utility functions for formatting data
        └── api.js                # API utility functions
```

## Key Features

### 1. **Hybrid Test Details Approach**
- **Quick Overview Modal** (`TestDetailModal.jsx`):
  - Essential test configuration (rate, workers, duration, distribution)
  - Quick results summary (total requests, success/failure counts, average latency)
  - "View Full Details & Charts" button for deeper analysis
  - Replay functionality for immediate actions
  - Focused on quick decision-making

- **Dedicated Details Page** (`TestDetailPage.jsx`):
  - Complete test analysis with tabbed interface (Overview, Results, Charts)
  - Comprehensive worker metrics with base64 decoding
  - Interactive charts (performance, latency distribution, status codes)
  - Detailed worker results table with all metrics (P50, P90, P95, P99, throughput)
  - Full error handling and status code analysis
  - URL-based access via `/test/{testId}`

### 2. **React Router Integration**
- **Client-side routing** with proper URL handling
- **Navbar navigation** using React Router Links
- **Seamless transitions** between modal and dedicated page
- **Direct URL access** to test details for sharing and bookmarking

### 3. **Advanced Metrics Handling**
- **Base64 decoding** of Vegeta metrics from multiple field names
- **Robust fallback logic** to raw API data when decoding fails
- **Vegeta format support** with proper unit conversions (nanoseconds to milliseconds)
- **Visual indicators** showing decoded/Vegeta format data
- **Comprehensive metrics display** including:
  - Latency percentiles (P50, P90, P95, P99)
  - Throughput and rate measurements
  - Bytes in/out statistics
  - Error counts and details
  - Status code aggregation and visualization

### 4. **DRY Principle Applied**
- **Shared Utilities**: Common formatting functions (`formatDate`, `formatDuration`, etc.) centralized in `utils/formatters.js`
- **Reusable Components**: UI components like `LoadingSpinner`, `StatusBadge` extracted and shared
- **API Functions**: All API calls centralized in `utils/api.js`
- **Constants**: Configuration and constants in `utils/constants.js`

### 5. **Separation of Concerns**
- **Context**: Authentication logic isolated in `AuthContext.jsx`
- **Components**: UI components organized by purpose (common, layout, modals)
- **Pages**: Page-level components handle routing and overall page logic
- **Utils**: Business logic and utilities separated from UI concerns

## Usage Examples

### React Router Navigation
```jsx
// App.jsx - Main routing setup
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { TestDetailPage } from './src/pages/TestDetailPage.jsx';

function App() {
    return (
        <Router>
            <Routes>
                <Route path="/" element={<DashboardPage />} />
                <Route path="/test-history" element={<TestHistoryPage />} />
                <Route path="/new-test" element={<NewTestPage />} />
                <Route path="/test/:testId" element={<TestDetailPage />} />
            </Routes>
        </Router>
    );
}
```

### Hybrid Modal Usage
```jsx
// TestHistoryPage.jsx - Using modal for quick view
import { TestDetailModal } from '../components/modals/TestDetailModal.jsx';

const TestHistoryPage = () => {
    const [selectedTestId, setSelectedTestId] = useState(null);

    return (
        <>
            {/* Test list with click handlers */}
            <button onClick={() => setSelectedTestId(test.id)}>
                View Details
            </button>

            {/* Quick overview modal */}
            <TestDetailModal
                testId={selectedTestId}
                isOpen={!!selectedTestId}
                onClose={() => setSelectedTestId(null)}
            />
        </>
    );
};
```

### Metrics Decoding
```jsx
// TestDetailPage.jsx - Base64 decoding with fallback
const decodeWorkerMetrics = (result) => {
    const base64Fields = ['metric', 'vegeta_metrics_base64', 'metrics_base64'];

    for (const field of base64Fields) {
        if (result[field]) {
            try {
                const decodedMetrics = JSON.parse(atob(result[field]));
                // Handle Vegeta format with unit conversions
                if (decodedMetrics.latencies) {
                    return {
                        ...result,
                        average_latency_ms: decodedMetrics.latencies.mean / 1000000,
                        p95_latency_ms: decodedMetrics.latencies['95th'] / 1000000,
                        success_rate: decodedMetrics.success,
                        decoded: true,
                        vegeta_format: true
                    };
                }
            } catch (e) {
                continue;
            }
        }
    }

    // Fallback to raw API data
    return { ...result, decoded: false };
};
```

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

1. **Optimal User Experience**:
   - Quick overview in modal for immediate insights
   - Detailed analysis in dedicated page for thorough investigation
   - Seamless navigation between both approaches

2. **Robust Data Handling**:
   - Advanced base64 decoding with multiple field name support
   - Proper Vegeta metric format handling with unit conversions
   - Fallback logic ensures data is always displayed

3. **Performance Optimized**:
   - Modal loads quickly with essential information
   - Full page loads comprehensive data only when needed
   - Charts and visualizations rendered efficiently

4. **Developer Friendly**:
   - Clear separation of concerns
   - Reusable components and utilities
   - Consistent patterns across the application

5. **Maintainability**:
   - Single responsibility principle applied
   - Shared logic reduces code duplication
   - Easy to test individual components

6. **Scalability**:
   - Modular architecture supports feature additions
   - React Router enables complex navigation patterns
   - Component library approach supports team development

7. **User-Centric Design**:
   - Quick access for decision-making via modal
   - Comprehensive analysis for detailed investigation
   - URL-based sharing for collaboration

## Technical Implementation

### Dependencies
- **react-router-dom**: Client-side routing
- **recharts**: Chart visualizations
- **lucide-react**: Icon library
- **react**: Core React framework

### Key Technical Features
- **Base64 Decoding**: Handles multiple field names and formats
- **Vegeta Metrics**: Proper parsing and unit conversion
- **Error Handling**: Robust fallback mechanisms
- **Visual Indicators**: Shows data source and format
- **Responsive Design**: Works across different screen sizes

## Migration Status

✅ **Phase 1**: Modular structure created
✅ **Phase 2**: React Router integration completed
✅ **Phase 3**: All main pages implemented (Dashboard, NewTest, TestHistory, TestDetail)
✅ **Phase 4**: Hybrid approach implemented with modal and dedicated page
✅ **Phase 5**: Advanced metrics handling with base64 decoding
✅ **Phase 6**: Chart visualizations and comprehensive UI

## Next Steps

1. **Testing**: Add unit tests for utility functions and components
2. **TypeScript**: Consider adding TypeScript for better type safety
3. **Performance**: Implement code splitting and lazy loading
4. **Monitoring**: Add error tracking and performance monitoring
5. **Documentation**: Create component documentation and usage guides
