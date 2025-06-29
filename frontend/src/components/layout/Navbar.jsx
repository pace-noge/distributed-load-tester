import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Rocket, History, LogOut, Zap } from 'lucide-react';
import { useAuth } from '../../contexts/AuthContext.jsx';

const NAV_ITEMS = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard, path: '/dashboard' },
    { id: 'new-test', label: 'New Test', icon: Rocket, path: '/new-test' },
    { id: 'test-history', label: 'Test History', icon: History, path: '/test-history' },
];

export const Navbar = () => {
    const { logout } = useAuth();
    const location = useLocation();

    return (
        <nav className="bg-white border-b border-gray-200 px-6 py-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center space-x-8">
                    <div className="flex items-center space-x-3">
                        <div className="h-8 w-8 bg-blue-600 rounded-lg flex items-center justify-center">
                            <Zap className="h-5 w-5 text-white" />
                        </div>
                        <h1 className="text-xl font-bold text-gray-900">Load Tester</h1>
                    </div>

                    <div className="flex space-x-1">
                        {NAV_ITEMS.map((item) => {
                            const Icon = item.icon;
                            const isActive = location.pathname === item.path;
                            return (
                                <Link
                                    key={item.id}
                                    to={item.path}
                                    className={`flex items-center space-x-2 px-4 py-2 rounded-lg font-medium transition-all ${
                                        isActive
                                            ? 'bg-blue-50 text-blue-700 border border-blue-200'
                                            : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
                                    }`}
                                >
                                    <Icon className="w-4 h-4" />
                                    <span>{item.label}</span>
                                </Link>
                            );
                        })}
                    </div>
                </div>

                <button
                    onClick={logout}
                    className="flex items-center space-x-2 text-gray-600 hover:text-red-600 px-3 py-2 rounded-lg hover:bg-red-50 transition-all"
                >
                    <LogOut className="w-4 h-4" />
                    <span>Logout</span>
                </button>
            </div>
        </nav>
    );
};
