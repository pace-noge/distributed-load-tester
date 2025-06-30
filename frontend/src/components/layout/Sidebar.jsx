import { useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
    LayoutDashboard,
    History,
    Rocket,
    BarChart3,
    Settings,
    User,
    Users,
    ChevronLeft,
    ChevronRight,
    Menu,
    X,
    Zap
} from 'lucide-react';
import { useAuth } from '../../contexts/AuthContext.jsx';

const navigationItems = [
    {
        name: 'Dashboard',
        path: '/dashboard',
        icon: LayoutDashboard,
        description: 'Overview and active tests'
    },
    {
        name: 'Test History',
        path: '/test-history',
        icon: History,
        description: 'View past test results'
    },
    {
        name: 'New Test',
        path: '/new-test',
        icon: Rocket,
        description: 'Create a new load test'
    },
    {
        name: 'Analytics',
        path: '/analytics',
        icon: BarChart3,
        description: 'Performance insights'
    }
];

const bottomNavigationItems = [
    {
        name: 'Settings',
        path: '/settings',
        icon: Settings,
        description: 'Application settings'
    },
    {
        name: 'Profile',
        path: '/profile',
        icon: User,
        description: 'User profile'
    }
];

export const Sidebar = ({ isOpen, setIsOpen, isCollapsed, setIsCollapsed }) => {
    const location = useLocation();
    const { logout, user } = useAuth();

    // Add user management for admin users
    const allNavigationItems = [
        ...navigationItems,
        ...(user?.role === 'admin' ? [{
            name: 'User Management',
            path: '/users',
            icon: Users,
            description: 'Manage user accounts'
        }] : [])
    ];

    const isActive = (path) => {
        if (path === '/dashboard' && location.pathname === '/') return true;
        return location.pathname === path;
    };

    const handleLogout = () => {
        logout();
    };

    return (
        <>
            {/* Mobile Overlay */}
            {isOpen && (
                <div
                    className="fixed inset-0 bg-black bg-opacity-50 z-40 lg:hidden"
                    onClick={() => setIsOpen(false)}
                />
            )}

            {/* Sidebar */}
            <div className={`
                fixed top-0 left-0 h-full bg-white border-r border-gray-200 z-50 transition-all duration-300 ease-in-out
                ${isOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
                ${isCollapsed ? 'lg:w-16' : 'lg:w-64'}
                w-64
            `}>
                <div className="flex flex-col h-full">
                    {/* Header */}
                    <div className="flex items-center justify-between p-4 border-b border-gray-200">
                        <div className={`flex items-center space-x-2 ${isCollapsed ? 'lg:justify-center' : ''}`}>
                            <div className="flex items-center justify-center w-8 h-8 bg-blue-600 rounded-lg">
                                <Zap className="w-5 h-5 text-white" />
                            </div>
                            {!isCollapsed && (
                                <div>
                                    <h1 className="text-lg font-bold text-gray-900">LoadTester</h1>
                                    <p className="text-xs text-gray-500">Distributed Testing</p>
                                </div>
                            )}
                        </div>

                        {/* Mobile Close Button */}
                        <button
                            onClick={() => setIsOpen(false)}
                            className="lg:hidden p-1 rounded-md hover:bg-gray-100"
                        >
                            <X className="w-5 h-5" />
                        </button>

                        {/* Desktop Collapse Button */}
                        <button
                            onClick={() => setIsCollapsed(!isCollapsed)}
                            className="hidden lg:block p-1 rounded-md hover:bg-gray-100"
                        >
                            {isCollapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
                        </button>
                    </div>

                    {/* Navigation */}
                    <nav className="flex-1 overflow-y-auto py-4">
                        <div className="px-3 space-y-1">
                            {allNavigationItems.map((item) => {
                                const Icon = item.icon;
                                const active = isActive(item.path);

                                return (
                                    <Link
                                        key={item.path}
                                        to={item.path}
                                        onClick={() => setIsOpen(false)}
                                        className={`
                                            flex items-center px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200
                                            ${active
                                                ? 'bg-blue-50 text-blue-700 border-r-2 border-blue-700'
                                                : 'text-gray-700 hover:bg-gray-50 hover:text-gray-900'
                                            }
                                            ${isCollapsed ? 'lg:justify-center lg:px-2' : ''}
                                        `}
                                        title={isCollapsed ? item.name : ''}
                                    >
                                        <Icon className={`
                                            w-5 h-5 flex-shrink-0
                                            ${active ? 'text-blue-700' : 'text-gray-500'}
                                            ${isCollapsed ? '' : 'mr-3'}
                                        `} />
                                        {!isCollapsed && (
                                            <div className="flex-1 min-w-0">
                                                <div className="truncate">{item.name}</div>
                                                <div className="text-xs text-gray-500 truncate">{item.description}</div>
                                            </div>
                                        )}
                                        {!isCollapsed && active && (
                                            <div className="w-2 h-2 bg-blue-600 rounded-full" />
                                        )}
                                    </Link>
                                );
                            })}
                        </div>

                        {/* Divider */}
                        <div className="my-4 mx-3 border-t border-gray-200" />

                        {/* Bottom Navigation */}
                        <div className="px-3 space-y-1">
                            {bottomNavigationItems.map((item) => {
                                const Icon = item.icon;
                                const active = isActive(item.path);

                                return (
                                    <Link
                                        key={item.path}
                                        to={item.path}
                                        onClick={() => setIsOpen(false)}
                                        className={`
                                            flex items-center px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200
                                            ${active
                                                ? 'bg-blue-50 text-blue-700 border-r-2 border-blue-700'
                                                : 'text-gray-700 hover:bg-gray-50 hover:text-gray-900'
                                            }
                                            ${isCollapsed ? 'lg:justify-center lg:px-2' : ''}
                                        `}
                                        title={isCollapsed ? item.name : ''}
                                    >
                                        <Icon className={`
                                            w-5 h-5 flex-shrink-0
                                            ${active ? 'text-blue-700' : 'text-gray-500'}
                                            ${isCollapsed ? '' : 'mr-3'}
                                        `} />
                                        {!isCollapsed && (
                                            <div className="flex-1 min-w-0">
                                                <div className="truncate">{item.name}</div>
                                                <div className="text-xs text-gray-500 truncate">{item.description}</div>
                                            </div>
                                        )}
                                    </Link>
                                );
                            })}
                        </div>
                    </nav>

                    {/* User Section */}
                    <div className="border-t border-gray-200 p-4">
                        <div className={`flex items-center ${isCollapsed ? 'lg:justify-center' : 'space-x-3'}`}>
                            <div className="flex-shrink-0">
                                <div className="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center">
                                    <User className="w-4 h-4 text-white" />
                                </div>
                            </div>
                            {!isCollapsed && (
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-medium text-gray-900 truncate">
                                        {user ? `${user.firstName} ${user.lastName}` : 'User'}
                                    </p>
                                    <p className="text-xs text-gray-500 truncate">
                                        {user ? `@${user.username} • ${user.role}` : 'Authenticated'}
                                    </p>
                                </div>
                            )}
                        </div>

                        {!isCollapsed && (
                            <button
                                onClick={handleLogout}
                                className="mt-3 w-full px-3 py-2 text-sm font-medium text-gray-700 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
                            >
                                Sign Out
                            </button>
                        )}
                    </div>
                </div>
            </div>
        </>
    );
};

export const MobileMenuButton = ({ onClick }) => (
    <button
        onClick={onClick}
        className="lg:hidden p-2 rounded-md text-gray-600 hover:text-gray-900 hover:bg-gray-100"
    >
        <Menu className="w-6 h-6" />
    </button>
);
