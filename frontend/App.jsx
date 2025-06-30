// Main App - Now using modular structure with React Router
import { useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';

// Context
import { AuthProvider, useAuth } from './src/contexts/AuthContext.jsx';

// Pages
import { LoginPage } from './src/pages/LoginPage.jsx';
import { TestHistoryPage } from './src/pages/TestHistoryPage.jsx';
import { DashboardPage } from './src/pages/DashboardPage.jsx';
import NewTestPage from './src/pages/NewTestPage.jsx';
import { TestDetailPage } from './src/pages/TestDetailPage.jsx';
import { UserManagementPage } from './src/pages/UserManagementPage.jsx';
import { ProfilePage } from './src/pages/ProfilePage.jsx';
import AnalyticsPage from './src/pages/AnalyticsPage.jsx';

// Layout Components
import { MainLayout } from './src/components/layout/MainLayout.jsx';

const AppContent = () => {
    const { isLoggedIn } = useAuth();

    // Handle auth errors by listening to custom events
    useEffect(() => {
        const handleAuthError = () => {
            localStorage.removeItem('jwt_token');
            window.location.reload();
        };

        window.addEventListener('auth-error', handleAuthError);
        return () => window.removeEventListener('auth-error', handleAuthError);
    }, []);

    if (!isLoggedIn) {
        return <LoginPage />;
    }

    return (
        <MainLayout>
            <Routes>
                <Route path="/" element={<Navigate to="/dashboard" replace />} />
                <Route path="/dashboard" element={<DashboardPage />} />
                <Route path="/new-test" element={<NewTestPage />} />
                <Route path="/test-history" element={<TestHistoryPage />} />
                <Route path="/test/:testId" element={<TestDetailPage />} />
                <Route path="/analytics" element={<AnalyticsPage />} />
                <Route path="/profile" element={<ProfilePage />} />
                <Route path="/users" element={<UserManagementPage />} />
            </Routes>
        </MainLayout>
    );
};

const App = () => {
    return (
        <Router>
            <AuthProvider>
                <AppContent />
            </AuthProvider>
        </Router>
    );
};

// Initialize React App
const container = document.getElementById('root');
const root = createRoot(container);
root.render(<App />);

export default App;
