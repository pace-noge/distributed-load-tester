import { useState, useEffect } from 'react';
import { Sidebar, MobileMenuButton } from './Sidebar.jsx';

export const MainLayout = ({ children }) => {
    const [sidebarOpen, setSidebarOpen] = useState(false);
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

    // Close sidebar on mobile when route changes
    useEffect(() => {
        setSidebarOpen(false);
    }, [children]);

    // Handle responsive behavior
    useEffect(() => {
        const handleResize = () => {
            if (window.innerWidth >= 1024) {
                setSidebarOpen(false); // Close mobile sidebar on desktop
            }
        };

        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, []);

    return (
        <div className="flex h-screen bg-gray-50">
            {/* Sidebar */}
            <Sidebar
                isOpen={sidebarOpen}
                setIsOpen={setSidebarOpen}
                isCollapsed={sidebarCollapsed}
                setIsCollapsed={setSidebarCollapsed}
            />

            {/* Main Content */}
            <div className={`
                flex-1 flex flex-col min-w-0 transition-all duration-300 ease-in-out
                ${sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-64'}
            `}>
                {/* Top Bar */}
                <header className="bg-white border-b border-gray-200 px-4 py-3 lg:px-6">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-4">
                            <MobileMenuButton onClick={() => setSidebarOpen(true)} />

                            {/* Breadcrumb or Page Title could go here */}
                            <div className="hidden sm:block">
                                <h2 className="text-lg font-semibold text-gray-900">
                                    {/* This will be dynamically set by each page */}
                                </h2>
                            </div>
                        </div>

                        {/* Right side actions could go here */}
                        <div className="flex items-center space-x-4">
                            {/* Notifications, user menu, etc. */}
                        </div>
                    </div>
                </header>

                {/* Page Content */}
                <main className="flex-1 overflow-auto">
                    <div className="p-4 lg:p-6">
                        {children}
                    </div>
                </main>
            </div>
        </div>
    );
};
