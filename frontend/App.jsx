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

// Layout Components
import { Navbar } from './src/components/layout/Navbar.jsx';

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
        <div className="min-h-screen bg-gray-50">
            <Navbar />
            <main className="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
                <Routes>
                    <Route path="/" element={<Navigate to="/dashboard" replace />} />
                    <Route path="/dashboard" element={<DashboardPage />} />
                    <Route path="/new-test" element={<NewTestPage />} />
                    <Route path="/test-history" element={<TestHistoryPage />} />
                    <Route path="/test/:testId" element={<TestDetailPage />} />
                </Routes>
            </main>
        </div>
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
