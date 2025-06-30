import { useState, useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';

// Context
import { AuthProvider, useAuth } from './src/contexts/AuthContext.jsx';

// Pages
import { LoginPage } from './src/pages/LoginPage.jsx';
import { TestHistoryPage } from './src/pages/TestHistoryPage.jsx';
import { InboxPage } from './src/pages/InboxPage.jsx';
import { SharedTestPage } from './src/pages/SharedTestPage.jsx';

// Layout Components
import { Navbar } from './src/components/layout/Navbar.jsx';

// Placeholder components for other pages
const DashboardPage = () => (
    <div className="text-center py-12">
        <h1 className="text-2xl font-bold text-gray-900 mb-4">Dashboard</h1>
        <p className="text-gray-600">Dashboard implementation coming soon...</p>
    </div>
);

const NewTestPage = () => (
    <div className="text-center py-12">
        <h1 className="text-2xl font-bold text-gray-900 mb-4">New Test</h1>
        <p className="text-gray-600">New test creation form coming soon...</p>
    </div>
);

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
        <Routes>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/dashboard" element={<DashboardPage />} />
            <Route path="/new-test" element={<NewTestPage />} />
            <Route path="/test-history" element={<TestHistoryPage />} />
            <Route path="/inbox" element={<InboxPage />} />
            <Route path="/shared/:linkId" element={<SharedTestPage />} />
            <Route path="/login" element={<LoginPage />} />
        </Routes>
    );
};

const App = () => (
    <AuthProvider>
        <Router>
            <Navbar />
            <main className="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
                <AppContent />
            </main>
        </Router>
    </AuthProvider>
);

// Initialize React App
const container = document.getElementById('root');
const root = createRoot(container);
root.render(<App />);
