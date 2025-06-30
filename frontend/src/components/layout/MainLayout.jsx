import { Navbar } from './Navbar.jsx';
import { Sidebar } from './Sidebar.jsx';
import { useState } from 'react';

export const MainLayout = ({ children }) => {
    const [sidebarOpen, setSidebarOpen] = useState(false);
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

    // Sidebar width: 64 (w-64) when expanded, 16 (w-16) when collapsed
    const sidebarWidth = sidebarCollapsed ? 'lg:w-16' : 'lg:w-64';
    const sidebarBaseWidth = 'w-64'; // for mobile

    return (
        <div className="flex flex-col min-h-screen bg-gray-50">
            {/* Top Bar */}
            <Navbar inboxRight />
            {/* Layout with Sidebar and Main Content */}
            <div className="flex flex-1 min-h-0">
                {/* Sidebar */}
                <div className={`hidden lg:block ${sidebarWidth} ${sidebarBaseWidth} flex-shrink-0 transition-all duration-300`}>
                    <Sidebar isOpen={sidebarOpen} setIsOpen={setSidebarOpen} isCollapsed={sidebarCollapsed} setIsCollapsed={setSidebarCollapsed} />
                </div>
                {/* Mobile Sidebar Overlay */}
                <div className="lg:hidden">
                    <Sidebar isOpen={sidebarOpen} setIsOpen={setSidebarOpen} isCollapsed={sidebarCollapsed} setIsCollapsed={setSidebarCollapsed} />
                </div>
                {/* Main Content */}
                <main className="flex-1 overflow-auto">
                    <div className="p-4 lg:p-6">
                        {children}
                    </div>
                </main>
            </div>
        </div>
    );
};
