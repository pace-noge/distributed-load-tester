// API Configuration
export const API_BASE_URL = 'http://localhost:8080/api';

// Color palette for charts
export const COLORS = ['#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#06B6D4'];

// Navigation items
export const NAV_ITEMS = [
    { id: 'dashboard', label: 'Dashboard', icon: 'LayoutDashboard' },
    { id: 'new-test', label: 'New Test', icon: 'Rocket' },
    { id: 'test-history', label: 'Test History', icon: 'History' },
];

// Test status configurations
export const STATUS_CONFIG = {
    COMPLETED: { bg: 'bg-green-100', text: 'text-green-800', icon: 'CheckCircle' },
    RUNNING: { bg: 'bg-blue-100', text: 'text-blue-800', icon: 'Clock' },
    FAILED: { bg: 'bg-red-100', text: 'text-red-800', icon: 'XCircle' },
    PENDING: { bg: 'bg-yellow-100', text: 'text-yellow-800', icon: 'AlertCircle' },
    DEFAULT: { bg: 'bg-gray-100', text: 'text-gray-800', icon: 'AlertCircle' }
};

// Modal tabs
export const MODAL_TABS = [
    { id: 'overview', label: 'Overview' },
    { id: 'results', label: 'Results' },
    { id: 'charts', label: 'Charts' }
];
