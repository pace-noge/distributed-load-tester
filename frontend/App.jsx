import React, { useState, useEffect, useCallback } from 'react';
import { createRoot } from 'react-dom/client';
import {
    ChevronRight, LayoutDashboard, Rocket, List, History, User, LogOut,
    Play, Eye, RefreshCw, ChevronLeft, BarChart3, TrendingUp, Zap,
    Clock, CheckCircle, XCircle, AlertCircle, Users, Target
} from 'lucide-react';
import {
    BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
    LineChart, Line, PieChart, Pie, Cell, Area, AreaChart
} from 'recharts';

const API_BASE_URL = 'http://localhost:8080/api';
const AUTH_URL = 'http://localhost:8080/auth/login';

// Color palette for charts
const COLORS = ['#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#06B6D4'];

// --- Auth Context ---
const AuthContext = React.createContext(null);

const AuthProvider = ({ children }) => {
    const [token, setToken] = useState(localStorage.getItem('jwt_token'));
    const [isLoggedIn, setIsLoggedIn] = useState(!!token);

    const login = (jwtToken) => {
        localStorage.setItem('jwt_token', jwtToken);
        setToken(jwtToken);
        setIsLoggedIn(true);
    };

    const logout = () => {
        localStorage.removeItem('jwt_token');
        setToken(null);
        setIsLoggedIn(false);
    };

    const getToken = () => token;

    return (
        <AuthContext.Provider value={{ isLoggedIn, login, logout, getToken }}>
            {children}
        </AuthContext.Provider>
    );
};

// --- API Utility ---
const authenticatedFetch = async (url, options = {}) => {
    const token = localStorage.getItem('jwt_token');
    if (!token) {
        throw new Error('Not authenticated');
    }

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        ...options.headers,
    };

    const response = await fetch(url, { ...options, headers });
    if (!response.ok) {
        if (response.status === 401 || response.status === 403) {
            const event = new CustomEvent('auth-error');
            window.dispatchEvent(event);
        }
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `API Error: ${response.status}`);
    }
    return response.json();
};

// --- Utility Components ---
const LoadingSpinner = ({ size = "default" }) => {
    const sizeClasses = {
        small: "h-4 w-4",
        default: "h-8 w-8",
        large: "h-12 w-12"
    };

    return (
        <div className={`animate-spin rounded-full ${sizeClasses[size]} border-b-2 border-blue-600`}></div>
    );
};

const StatusBadge = ({ status, className = "" }) => {
    const getStatusConfig = (status) => {
        switch (status?.toUpperCase()) {
            case 'COMPLETED':
                return { bg: 'bg-green-100', text: 'text-green-800', icon: CheckCircle };
            case 'RUNNING':
                return { bg: 'bg-blue-100', text: 'text-blue-800', icon: Clock };
            case 'FAILED':
                return { bg: 'bg-red-100', text: 'text-red-800', icon: XCircle };
            case 'PENDING':
                return { bg: 'bg-yellow-100', text: 'text-yellow-800', icon: AlertCircle };
            default:
                return { bg: 'bg-gray-100', text: 'text-gray-800', icon: AlertCircle };
        }
    };

    const config = getStatusConfig(status);
    const Icon = config.icon;

    return (
        <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.bg} ${config.text} ${className}`}>
            <Icon className="w-3 h-3 mr-1" />
            {status}
        </span>
    );
};

const Pagination = ({ currentPage, totalPages, onPageChange, className = "" }) => {
    const getPageNumbers = () => {
        const pages = [];
        const showEllipsis = totalPages > 7;

        if (!showEllipsis) {
            for (let i = 1; i <= totalPages; i++) {
                pages.push(i);
            }
        } else {
            if (currentPage <= 4) {
                for (let i = 1; i <= 5; i++) pages.push(i);
                pages.push('...');
                pages.push(totalPages);
            } else if (currentPage >= totalPages - 3) {
                pages.push(1);
                pages.push('...');
                for (let i = totalPages - 4; i <= totalPages; i++) pages.push(i);
            } else {
                pages.push(1);
                pages.push('...');
                for (let i = currentPage - 1; i <= currentPage + 1; i++) pages.push(i);
                pages.push('...');
                pages.push(totalPages);
            }
        }

        return pages;
    };

    return (
        <div className={`flex items-center justify-between ${className}`}>
            <div className="flex items-center space-x-2">
                <button
                    onClick={() => onPageChange(Math.max(1, currentPage - 1))}
                    disabled={currentPage === 1}
                    className="px-3 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    <ChevronLeft className="w-4 h-4" />
                </button>

                {getPageNumbers().map((page, index) => (
                    <button
                        key={index}
                        onClick={() => typeof page === 'number' && onPageChange(page)}
                        disabled={page === '...'}
                        className={`px-3 py-2 text-sm font-medium rounded-md ${
                            page === currentPage
                                ? 'bg-blue-600 text-white'
                                : page === '...'
                                ? 'text-gray-400 cursor-default'
                                : 'text-gray-700 bg-white border border-gray-300 hover:bg-gray-50'
                        }`}
                    >
                        {page}
                    </button>
                ))}

                <button
                    onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
                    disabled={currentPage === totalPages}
                    className="px-3 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    <ChevronRight className="w-4 h-4" />
                </button>
            </div>

            <span className="text-sm text-gray-700">
                Page {currentPage} of {totalPages}
            </span>
        </div>
    );
};

// --- Login Page ---
const LoginPage = () => {
    const [username, setUsername] = useState('admin');
    const [password, setPassword] = useState('password');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const { login } = React.useContext(AuthContext);

    const handleLogin = async (e) => {
        e.preventDefault();
        setError('');
        setLoading(true);
        try {
            const response = await fetch(AUTH_URL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password }),
            });

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({ message: 'Login failed' }));
                throw new Error(errorData.message || 'Login failed');
            }

            const data = await response.json();
            login(data.token);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
            <div className="bg-white p-8 rounded-2xl shadow-2xl w-full max-w-md">
                <div className="text-center mb-8">
                    <div className="mx-auto h-12 w-12 bg-blue-600 rounded-xl flex items-center justify-center mb-4">
                        <Zap className="h-6 w-6 text-white" />
                    </div>
                    <h2 className="text-3xl font-bold text-gray-900">Welcome Back</h2>
                    <p className="text-gray-600 mt-2">Sign in to your load tester dashboard</p>
                </div>

                <form onSubmit={handleLogin} className="space-y-6">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Username</label>
                        <input
                            type="text"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                            required
                            disabled={loading}
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Password</label>
                        <input
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                            required
                            disabled={loading}
                        />
                    </div>

                    {error && (
                        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
                            {error}
                        </div>
                    )}

                    <button
                        type="submit"
                        className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 flex items-center justify-center transition-all font-medium"
                        disabled={loading}
                    >
                        {loading ? <LoadingSpinner size="small" /> : 'Sign In'}
                    </button>
                </form>
            </div>
        </div>
    );
};

// --- Navigation ---
const Navbar = ({ currentPage, setCurrentPage }) => {
    const { logout } = React.useContext(AuthContext);

    const navItems = [
        { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
        { id: 'new-test', label: 'New Test', icon: Rocket },
        { id: 'test-history', label: 'Test History', icon: History },
    ];

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
                        {navItems.map((item) => {
                            const Icon = item.icon;
                            return (
                                <button
                                    key={item.id}
                                    onClick={() => setCurrentPage(item.id)}
                                    className={`flex items-center space-x-2 px-4 py-2 rounded-lg font-medium transition-all ${
                                        currentPage === item.id
                                            ? 'bg-blue-50 text-blue-700 border border-blue-200'
                                            : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
                                    }`}
                                >
                                    <Icon className="w-4 h-4" />
                                    <span>{item.label}</span>
                                </button>
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

// --- Test Detail Modal ---
const TestDetailModal = ({ testId, isOpen, onClose }) => {
    const [testDetail, setTestDetail] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [activeTab, setActiveTab] = useState('overview');

    const fetchTestDetail = useCallback(async () => {
        if (!testId || !isOpen) return;

        setLoading(true);
        setError('');
        try {
            const data = await authenticatedFetch(`${API_BASE_URL}/tests/${testId}`);
            setTestDetail(data);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }, [testId, isOpen]);

    useEffect(() => {
        fetchTestDetail();
    }, [fetchTestDetail]);

    const handleReplay = async () => {
        try {
            await authenticatedFetch(`${API_BASE_URL}/tests/${testId}/replay`, {
                method: 'POST',
                body: JSON.stringify({ name: `${testDetail?.test?.name} (Replay)` })
            });
            alert('Test replayed successfully!');
            onClose();
        } catch (err) {
            alert(`Failed to replay test: ${err.message}`);
        }
    };

    if (!isOpen) return null;

    const tabs = [
        { id: 'overview', label: 'Overview' },
        { id: 'results', label: 'Results' },
        { id: 'charts', label: 'Charts' }
    ];

    // Prepare chart data
    const workerPerformanceData = testDetail?.results?.map((result, index) => ({
        name: `Worker ${index + 1}`,
        requests: result.total_requests,
        completed: result.completed_requests,
        avgLatency: result.average_latency_ms,
        p95Latency: result.p95_latency_ms,
        successRate: result.success_rate * 100
    })) || [];

    const statusCodeData = testDetail?.aggregated_result ?
        Object.entries(testDetail.aggregated_result.error_rates || {}).map(([code, count]) => ({
            name: `HTTP ${code}`,
            value: count
        })) : [];

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-2xl shadow-2xl w-full max-w-6xl max-h-[90vh] overflow-hidden">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-gray-200">
                    <div>
                        <h2 className="text-2xl font-bold text-gray-900">Test Details</h2>
                        {testDetail?.test && (
                            <p className="text-gray-600 mt-1">{testDetail.test.name}</p>
                        )}
                    </div>
                    <div className="flex items-center space-x-3">
                        <button
                            onClick={handleReplay}
                            className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-all"
                        >
                            <Play className="w-4 h-4" />
                            <span>Replay</span>
                        </button>
                        <button
                            onClick={onClose}
                            className="text-gray-400 hover:text-gray-600 p-2"
                        >
                            <XCircle className="w-6 h-6" />
                        </button>
                    </div>
                </div>

                {/* Tabs */}
                <div className="border-b border-gray-200">
                    <nav className="flex space-x-8 px-6">
                        {tabs.map((tab) => (
                            <button
                                key={tab.id}
                                onClick={() => setActiveTab(tab.id)}
                                className={`py-4 text-sm font-medium border-b-2 transition-all ${
                                    activeTab === tab.id
                                        ? 'border-blue-500 text-blue-600'
                                        : 'border-transparent text-gray-500 hover:text-gray-700'
                                }`}
                            >
                                {tab.label}
                            </button>
                        ))}
                    </nav>
                </div>

                {/* Content */}
                <div className="p-6 overflow-y-auto max-h-[60vh]">
                    {loading ? (
                        <div className="flex items-center justify-center py-12">
                            <LoadingSpinner size="large" />
                        </div>
                    ) : error ? (
                        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
                            {error}
                        </div>
                    ) : !testDetail ? (
                        <div className="text-gray-500 text-center py-12">No data available</div>
                    ) : (
                        <>
                            {activeTab === 'overview' && (
                                <div className="space-y-6">
                                    {/* Test Info */}
                                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                                        <div className="bg-blue-50 p-4 rounded-lg">
                                            <div className="flex items-center space-x-3">
                                                <Target className="w-8 h-8 text-blue-600" />
                                                <div>
                                                    <div className="text-2xl font-bold text-blue-900">
                                                        {testDetail.test.rate_per_second}
                                                    </div>
                                                    <div className="text-sm text-blue-700">Req/sec</div>
                                                </div>
                                            </div>
                                        </div>

                                        <div className="bg-green-50 p-4 rounded-lg">
                                            <div className="flex items-center space-x-3">
                                                <Users className="w-8 h-8 text-green-600" />
                                                <div>
                                                    <div className="text-2xl font-bold text-green-900">
                                                        {testDetail.test.worker_count}
                                                    </div>
                                                    <div className="text-sm text-green-700">Workers</div>
                                                </div>
                                            </div>
                                        </div>

                                        <div className="bg-yellow-50 p-4 rounded-lg">
                                            <div className="flex items-center space-x-3">
                                                <Clock className="w-8 h-8 text-yellow-600" />
                                                <div>
                                                    <div className="text-2xl font-bold text-yellow-900">
                                                        {testDetail.test.duration_seconds}
                                                    </div>
                                                    <div className="text-sm text-yellow-700">Duration</div>
                                                </div>
                                            </div>
                                        </div>

                                        <div className="bg-purple-50 p-4 rounded-lg">
                                            <div className="flex items-center space-x-3">
                                                <BarChart3 className="w-8 h-8 text-purple-600" />
                                                <div>
                                                    <div className="text-2xl font-bold text-purple-900">
                                                        {testDetail.test.rate_distribution}
                                                    </div>
                                                    <div className="text-sm text-purple-700">Distribution</div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                    {/* Status and Timeline */}
                                    <div className="bg-gray-50 p-6 rounded-lg">
                                        <h3 className="text-lg font-medium text-gray-900 mb-4">Test Status</h3>
                                        <div className="flex items-center justify-between">
                                            <StatusBadge status={testDetail.test.status} />
                                            <div className="text-sm text-gray-600">
                                                Created: {new Date(testDetail.test.created_at).toLocaleString()}
                                            </div>
                                        </div>

                                        {testDetail.test.assigned_workers_ids?.length > 0 && (
                                            <div className="mt-4">
                                                <div className="text-sm font-medium text-gray-700 mb-2">Worker Status:</div>
                                                <div className="flex flex-wrap gap-2">
                                                    {testDetail.test.assigned_workers_ids.map((workerId) => (
                                                        <span key={workerId} className="px-2 py-1 bg-blue-100 text-blue-800 rounded text-xs">
                                                            {workerId}
                                                        </span>
                                                    ))}
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            )}

                            {activeTab === 'results' && (
                                <div className="space-y-6">
                                    {testDetail.aggregated_result ? (
                                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                                            <div className="bg-blue-50 p-4 rounded-lg">
                                                <div className="text-2xl font-bold text-blue-900">
                                                    {testDetail.aggregated_result.total_requests.toLocaleString()}
                                                </div>
                                                <div className="text-sm text-blue-700">Total Requests</div>
                                            </div>

                                            <div className="bg-green-50 p-4 rounded-lg">
                                                <div className="text-2xl font-bold text-green-900">
                                                    {testDetail.aggregated_result.successful_requests.toLocaleString()}
                                                </div>
                                                <div className="text-sm text-green-700">Successful</div>
                                            </div>

                                            <div className="bg-red-50 p-4 rounded-lg">
                                                <div className="text-2xl font-bold text-red-900">
                                                    {testDetail.aggregated_result.failed_requests.toLocaleString()}
                                                </div>
                                                <div className="text-sm text-red-700">Failed</div>
                                            </div>

                                            <div className="bg-yellow-50 p-4 rounded-lg">
                                                <div className="text-2xl font-bold text-yellow-900">
                                                    {testDetail.aggregated_result.avg_latency_ms.toFixed(2)}ms
                                                </div>
                                                <div className="text-sm text-yellow-700">Avg Latency</div>
                                            </div>
                                        </div>
                                    ) : (
                                        <div className="text-gray-500 text-center py-8">
                                            No aggregated results available yet.
                                        </div>
                                    )}

                                    {/* Individual Worker Results */}
                                    {testDetail.results?.length > 0 && (
                                        <div className="overflow-x-auto">
                                            <table className="min-w-full divide-y divide-gray-200">
                                                <thead className="bg-gray-50">
                                                    <tr>
                                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Worker</th>
                                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Requests</th>
                                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Success Rate</th>
                                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Avg Latency</th>
                                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">P95 Latency</th>
                                                    </tr>
                                                </thead>
                                                <tbody className="bg-white divide-y divide-gray-200">
                                                    {testDetail.results.map((result, index) => (
                                                        <tr key={result.id}>
                                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                                                                Worker {index + 1}
                                                            </td>
                                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                                {result.completed_requests}/{result.total_requests}
                                                            </td>
                                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                                {(result.success_rate * 100).toFixed(1)}%
                                                            </td>
                                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                                {result.average_latency_ms.toFixed(2)}ms
                                                            </td>
                                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                                {result.p95_latency_ms.toFixed(2)}ms
                                                            </td>
                                                        </tr>
                                                    ))}
                                                </tbody>
                                            </table>
                                        </div>
                                    )}
                                </div>
                            )}

                            {activeTab === 'charts' && (
                                <div className="space-y-8">
                                    {workerPerformanceData.length > 0 && (
                                        <>
                                            {/* Worker Performance Chart */}
                                            <div className="bg-white p-6 rounded-lg border border-gray-200">
                                                <h3 className="text-lg font-medium text-gray-900 mb-4">Worker Performance</h3>
                                                <ResponsiveContainer width="100%" height={300}>
                                                    <BarChart data={workerPerformanceData}>
                                                        <CartesianGrid strokeDasharray="3 3" />
                                                        <XAxis dataKey="name" />
                                                        <YAxis />
                                                        <Tooltip />
                                                        <Legend />
                                                        <Bar dataKey="requests" fill="#3B82F6" name="Total Requests" />
                                                        <Bar dataKey="completed" fill="#10B981" name="Completed Requests" />
                                                    </BarChart>
                                                </ResponsiveContainer>
                                            </div>

                                            {/* Latency Chart */}
                                            <div className="bg-white p-6 rounded-lg border border-gray-200">
                                                <h3 className="text-lg font-medium text-gray-900 mb-4">Latency Distribution</h3>
                                                <ResponsiveContainer width="100%" height={300}>
                                                    <LineChart data={workerPerformanceData}>
                                                        <CartesianGrid strokeDasharray="3 3" />
                                                        <XAxis dataKey="name" />
                                                        <YAxis />
                                                        <Tooltip />
                                                        <Legend />
                                                        <Line type="monotone" dataKey="avgLatency" stroke="#F59E0B" name="Avg Latency (ms)" />
                                                        <Line type="monotone" dataKey="p95Latency" stroke="#EF4444" name="P95 Latency (ms)" />
                                                    </LineChart>
                                                </ResponsiveContainer>
                                            </div>
                                        </>
                                    )}

                                    {/* Status Code Distribution */}
                                    {statusCodeData.length > 0 && (
                                        <div className="bg-white p-6 rounded-lg border border-gray-200">
                                            <h3 className="text-lg font-medium text-gray-900 mb-4">Status Code Distribution</h3>
                                            <ResponsiveContainer width="100%" height={300}>
                                                <PieChart>
                                                    <Pie
                                                        data={statusCodeData}
                                                        cx="50%"
                                                        cy="50%"
                                                        labelLine={false}
                                                        label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                                                        outerRadius={80}
                                                        fill="#8884d8"
                                                        dataKey="value"
                                                    >
                                                        {statusCodeData.map((entry, index) => (
                                                            <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                                                        ))}
                                                    </Pie>
                                                    <Tooltip />
                                                </PieChart>
                                            </ResponsiveContainer>
                                        </div>
                                    )}

                                    {workerPerformanceData.length === 0 && statusCodeData.length === 0 && (
                                        <div className="text-gray-500 text-center py-12">
                                            No chart data available. Results may still be processing.
                                        </div>
                                    )}
                                </div>
                            )}
                        </>
                    )}
                </div>
            </div>
        </div>
    );
};

// Continue in next part...
// Continuation of App_Enhanced.jsx

// --- Test History Page ---
const TestHistoryPage = () => {
    const [tests, setTests] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [selectedTest, setSelectedTest] = useState(null);
    const [currentPage, setCurrentPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [pageSize] = useState(10);

    const fetchTestHistory = useCallback(async (page = 1) => {
        setLoading(true);
        setError('');
        try {
            const data = await authenticatedFetch(`${API_BASE_URL}/tests`);
            if (Array.isArray(data)) { setTests(data); setTotalPages(1); } else { setTests(data.tests || data.data || []); setTotalPages(data.total_pages || 1); }
            setTotalPages(data.total_pages || 1);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }, [pageSize]);

    useEffect(() => {
        fetchTestHistory(currentPage);
    }, [currentPage, fetchTestHistory]);

    const handlePageChange = (page) => {
        setCurrentPage(page);
    };

    const handleReplay = async (testId, testName) => {
        try {
            await authenticatedFetch(`${API_BASE_URL}/tests/${testId}/replay`, {
                method: 'POST',
                body: JSON.stringify({ name: `${testName} (Replay)` })
            });
            alert('Test replayed successfully!');
            fetchTestHistory(currentPage);
        } catch (err) {
            alert(`Failed to replay test: ${err.message}`);
        }
    };

    const formatDate = (dateString) => {
        return new Date(dateString).toLocaleString();
    };

    const getStatusColor = (status) => {
        switch (status?.toUpperCase()) {
            case 'COMPLETED': return 'text-green-600 bg-green-100';
            case 'RUNNING': return 'text-blue-600 bg-blue-100';
            case 'FAILED': return 'text-red-600 bg-red-100';
            case 'PENDING': return 'text-yellow-600 bg-yellow-100';
            default: return 'text-gray-600 bg-gray-100';
        }
    };

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-gray-900">Test History</h1>
                    <p className="text-gray-600 mt-2">View and manage your load test history</p>
                </div>
                <button
                    onClick={() => fetchTestHistory(currentPage)}
                    className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-all"
                >
                    <RefreshCw className="w-4 h-4" />
                    <span>Refresh</span>
                </button>
            </div>

            {error && (
                <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
                    {error}
                </div>
            )}

            {loading ? (
                <div className="flex items-center justify-center py-12">
                    <LoadingSpinner size="large" />
                </div>
            ) : tests.length === 0 ? (
                <div className="text-center py-12">
                    <History className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                    <h3 className="text-lg font-medium text-gray-900 mb-2">No tests found</h3>
                    <p className="text-gray-600">You haven't run any load tests yet.</p>
                </div>
            ) : (
                <>
                    <div className="bg-white shadow-sm rounded-lg overflow-hidden">
                        <div className="overflow-x-auto">
                            <table className="min-w-full divide-y divide-gray-200">
                                <thead className="bg-gray-50">
                                    <tr>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Test Name</th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Configuration</th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                                    </tr>
                                </thead>
                                <tbody className="bg-white divide-y divide-gray-200">
                                    {tests.map((test) => (
                                        <tr key={test.id} className="hover:bg-gray-50">
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <div>
                                                    <div className="text-sm font-medium text-gray-900">{test.name}</div>
                                                    <div className="text-sm text-gray-500">{test.id}</div>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <StatusBadge status={test.status} />
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                <div className="space-y-1">
                                                    <div>{test.rate_per_second} req/s × {test.worker_count} workers</div>
                                                    <div className="text-xs text-gray-400">
                                                        {test.duration_seconds} • {test.rate_distribution}
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                {formatDate(test.createdAt || test.created_at)}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
                                                <button
                                                    onClick={() => setSelectedTest(test.id)}
                                                    className="text-blue-600 hover:text-blue-900 flex items-center space-x-1"
                                                >
                                                    <Eye className="w-4 h-4" />
                                                    <span>View</span>
                                                </button>
                                                <button
                                                    onClick={() => handleReplay(test.id, test.name)}
                                                    className="text-green-600 hover:text-green-900 flex items-center space-x-1"
                                                >
                                                    <Play className="w-4 h-4" />
                                                    <span>Replay</span>
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    </div>

                    {totalPages > 1 && (
                        <Pagination
                            currentPage={currentPage}
                            totalPages={totalPages}
                            onPageChange={handlePageChange}
                            className="mt-6"
                        />
                    )}
                </>
            )}

            <TestDetailModal
                testId={selectedTest}
                isOpen={!!selectedTest}
                onClose={() => setSelectedTest(null)}
            />
        </div>
    );
};

// --- Dashboard Page ---
const DashboardPage = () => {
    const [dashboardData, setDashboardData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [connectionStatus, setConnectionStatus] = useState('connecting');
    const [usingWebSocket, setUsingWebSocket] = useState(false);
    const { getToken } = React.useContext(AuthContext);

    const fetchDashboardData = useCallback(async () => {
        try {
            const data = await authenticatedFetch(`${API_BASE_URL}/dashboard`);
            setDashboardData(data);
            setError('');
            setLoading(false);
        } catch (err) {
            setError(`Failed to load dashboard: ${err.message}`);
        }
    }, []);

    useEffect(() => {
        const token = getToken();
        if (!token) {
            setError('Authentication required');
            setLoading(false);
            return;
        }

        // Try WebSocket first, fallback to HTTP polling
        const wsUrl = `ws://localhost:8080/ws?token=${encodeURIComponent(token)}`;
        const ws = new WebSocket(wsUrl);
        let wsConnected = false;

        const fallbackTimer = setTimeout(() => {
            if (!wsConnected) {
                setUsingWebSocket(false);
                setConnectionStatus('http_polling');
                fetchDashboardData();
                const interval = setInterval(fetchDashboardData, 5000);
                return () => clearInterval(interval);
            }
        }, 5000);

        ws.onopen = () => {
            wsConnected = true;
            clearTimeout(fallbackTimer);
            setConnectionStatus('connected');
            setUsingWebSocket(true);
            setError('');
        };

        ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                if (message.type === 'dashboard_update') {
                    setDashboardData(message.data);
                    setLoading(false);
                    setError('');
                }
            } catch (err) {
                setError('Error parsing server message');
            }
        };

        ws.onclose = (event) => {
            setConnectionStatus('disconnected');
            if (event.code !== 1000) {
                setError(`Connection lost. Falling back to HTTP polling...`);
                setUsingWebSocket(false);
                fetchDashboardData();
                const interval = setInterval(fetchDashboardData, 5000);
                setTimeout(() => clearInterval(interval), 60000);
            }
        };

        ws.onerror = () => {
            setConnectionStatus('error');
            setError('WebSocket connection failed. Falling back to HTTP polling...');
            setUsingWebSocket(false);
            clearTimeout(fallbackTimer);
            fetchDashboardData();
            const interval = setInterval(fetchDashboardData, 5000);
            setTimeout(() => clearInterval(interval), 60000);
        };

        return () => {
            clearTimeout(fallbackTimer);
            if (ws.readyState === WebSocket.OPEN) {
                ws.close();
            }
        };
    }, [getToken, fetchDashboardData]);

    if (loading) {
        return (
            <div className="flex items-center justify-center py-12">
                <LoadingSpinner size="large" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-gray-900">Dashboard</h1>
                    <p className="text-gray-600 mt-2">Monitor your load testing infrastructure</p>
                </div>
                <div className="flex items-center space-x-4">
                    <div className={`flex items-center space-x-2 px-3 py-1 rounded-full text-sm ${
                        connectionStatus === 'connected' ? 'bg-green-100 text-green-700' :
                        connectionStatus === 'http_polling' ? 'bg-yellow-100 text-yellow-700' :
                        'bg-red-100 text-red-700'
                    }`}>
                        <div className={`w-2 h-2 rounded-full ${
                            connectionStatus === 'connected' ? 'bg-green-500' :
                            connectionStatus === 'http_polling' ? 'bg-yellow-500' :
                            'bg-red-500'
                        }`}></div>
                        <span>{usingWebSocket ? 'WebSocket' : 'HTTP Polling'}</span>
                    </div>
                </div>
            </div>

            {error && (
                <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
                    {error}
                </div>
            )}

            {dashboardData && (
                <>
                    {/* Worker Status Cards */}
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                        <div className="bg-blue-50 p-6 rounded-lg border border-blue-200">
                            <div className="flex items-center space-x-4">
                                <div className="p-3 bg-blue-600 rounded-lg">
                                    <Users className="w-6 h-6 text-white" />
                                </div>
                                <div>
                                    <div className="text-2xl font-bold text-blue-900">
                                        {dashboardData.total_workers}
                                    </div>
                                    <div className="text-sm text-blue-700">Total Workers</div>
                                </div>
                            </div>
                        </div>

                        <div className="bg-green-50 p-6 rounded-lg border border-green-200">
                            <div className="flex items-center space-x-4">
                                <div className="p-3 bg-green-600 rounded-lg">
                                    <CheckCircle className="w-6 h-6 text-white" />
                                </div>
                                <div>
                                    <div className="text-2xl font-bold text-green-900">
                                        {dashboardData.available_workers}
                                    </div>
                                    <div className="text-sm text-green-700">Available</div>
                                </div>
                            </div>
                        </div>

                        <div className="bg-yellow-50 p-6 rounded-lg border border-yellow-200">
                            <div className="flex items-center space-x-4">
                                <div className="p-3 bg-yellow-600 rounded-lg">
                                    <Clock className="w-6 h-6 text-white" />
                                </div>
                                <div>
                                    <div className="text-2xl font-bold text-yellow-900">
                                        {dashboardData.busy_workers}
                                    </div>
                                    <div className="text-sm text-yellow-700">Busy</div>
                                </div>
                            </div>
                        </div>
                    </div>

                    {/* Active Tests */}
                    {dashboardData.active_tests && dashboardData.active_tests.length > 0 && (
                        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                            <h2 className="text-xl font-semibold text-gray-900 mb-4">Active Tests</h2>
                            <div className="space-y-4">
                                {dashboardData.active_tests.map((test) => (
                                    <div key={test.test_id} className="border border-gray-200 rounded-lg p-4">
                                        <div className="flex items-center justify-between mb-3">
                                            <div>
                                                <h3 className="font-medium text-gray-900">{test.test_name}</h3>
                                                <p className="text-sm text-gray-600">{test.test_id}</p>
                                            </div>
                                            <StatusBadge status={test.status} />
                                        </div>

                                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-3">
                                            <div className="text-center">
                                                <div className="text-lg font-semibold text-gray-900">
                                                    {test.assigned_workers}
                                                </div>
                                                <div className="text-xs text-gray-600">Assigned</div>
                                            </div>
                                            <div className="text-center">
                                                <div className="text-lg font-semibold text-green-600">
                                                    {test.completed_workers}
                                                </div>
                                                <div className="text-xs text-gray-600">Completed</div>
                                            </div>
                                            <div className="text-center">
                                                <div className="text-lg font-semibold text-blue-600">
                                                    {test.total_requests_sent.toLocaleString()}
                                                </div>
                                                <div className="text-xs text-gray-600">Requests Sent</div>
                                            </div>
                                            <div className="text-center">
                                                <div className="text-lg font-semibold text-purple-600">
                                                    {(test.progress * 100).toFixed(1)}%
                                                </div>
                                                <div className="text-xs text-gray-600">Progress</div>
                                            </div>
                                        </div>

                                        <div className="w-full bg-gray-200 rounded-full h-2">
                                            <div
                                                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                                                style={{ width: `${test.progress * 100}%` }}
                                            ></div>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Worker Summaries */}
                    {dashboardData.worker_summaries && dashboardData.worker_summaries.length > 0 && (
                        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                            <h2 className="text-xl font-semibold text-gray-900 mb-4">Worker Status</h2>
                            <div className="overflow-x-auto">
                                <table className="min-w-full divide-y divide-gray-200">
                                    <thead className="bg-gray-50">
                                        <tr>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Worker ID</th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Current Test</th>
                                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Progress</th>
                                        </tr>
                                    </thead>
                                    <tbody className="bg-white divide-y divide-gray-200">
                                        {dashboardData.worker_summaries.map((worker) => (
                                            <tr key={worker.worker_id}>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                                                    {worker.worker_id}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap">
                                                    <StatusBadge status={worker.status_type} />
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                    {worker.current_test_id || 'None'}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                    {worker.total_requests > 0 ? (
                                                        <div className="flex items-center space-x-2">
                                                            <div className="flex-1 bg-gray-200 rounded-full h-2">
                                                                <div
                                                                    className="bg-blue-600 h-2 rounded-full"
                                                                    style={{
                                                                        width: `${(worker.completed_requests / worker.total_requests) * 100}%`
                                                                    }}
                                                                ></div>
                                                            </div>
                                                            <span className="text-xs">
                                                                {worker.completed_requests}/{worker.total_requests}
                                                            </span>
                                                        </div>
                                                    ) : (
                                                        'Idle'
                                                    )}
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    )}
                </>
            )}
        </div>
    );
};

// Continue to next part...
// Final part of App_Enhanced.jsx

// --- New Test Page ---
const NewTestPage = () => {
    const [formData, setFormData] = useState({
        name: '',
        durationSeconds: '30s',
        ratePerSecond: 10,
        workerCount: 1,
        rateDistribution: 'shared',
        rateWeights: [],
        rampDuration: '',
        rampStartDelay: '',
        rampSteps: '',
        targets: '[{"method":"GET","url":"https://httpbin.org/get"}]',
        // Vegeta options
        timeout: 30,
        redirects: 10,
        keepalive: true,
        http2: false,
        insecure: false,
        connections: 10000,
        vegetaPayloadJson: '{}'
    });

    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    const handleInputChange = (e) => {
        const { name, value, type, checked } = e.target;
        setFormData(prev => ({
            ...prev,
            [name]: type === 'checkbox' ? checked : type === 'number' ? Number(value) : value
        }));
    };

    const handleWeightsChange = (index, value) => {
        const newWeights = [...formData.rateWeights];
        newWeights[index] = parseFloat(value) || 0;
        setFormData(prev => ({ ...prev, rateWeights: newWeights }));
    };

    const updateWorkerCount = (count) => {
        const weights = Array(count).fill(1.0);
        setFormData(prev => ({
            ...prev,
            workerCount: count,
            rateWeights: prev.rateDistribution === 'weighted' ? weights : []
        }));
    };

    const handleRateDistributionChange = (distribution) => {
        const weights = distribution === 'weighted' ? Array(formData.workerCount).fill(1.0) : [];
        setFormData(prev => ({
            ...prev,
            rateDistribution: distribution,
            rateWeights: weights
        }));
    };

    const buildVegetaPayload = () => {
        const baseOptions = {
            timeout: formData.timeout,
            redirects: formData.redirects,
            keepalive: formData.keepalive,
            http2: formData.http2,
            insecure: formData.insecure,
            connections: formData.connections,
        };

        let userOptions = {};
        try {
            if (formData.vegetaPayloadJson.trim()) {
                userOptions = JSON.parse(formData.vegetaPayloadJson);
            }
        } catch (e) {
            // If JSON is invalid, use empty object
        }

        // Merge with user options taking precedence
        return JSON.stringify({ ...baseOptions, ...userOptions });
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setSubmitting(true);
        setError('');
        setSuccess('');

        try {
            // Validate targets JSON
            JSON.parse(formData.targets);

            // Build final payload
            const payload = {
                name: formData.name,
                duration_seconds: formData.durationSeconds,
                rate_per_second: formData.ratePerSecond,
                worker_count: formData.workerCount,
                rate_distribution: formData.rateDistribution,
                targets_base64: btoa(formData.targets),
                vegeta_payload_json: buildVegetaPayload(),
            };

            // Add conditional fields
            if (formData.rateDistribution === 'weighted' && formData.rateWeights.length > 0) {
                payload.rate_weights = formData.rateWeights;
            }

            if (formData.rateDistribution === 'ramped') {
                if (formData.rampDuration) payload.ramp_duration = formData.rampDuration;
                if (formData.rampStartDelay) payload.ramp_start_delay = formData.rampStartDelay;
                if (formData.rampSteps) payload.ramp_steps = parseInt(formData.rampSteps);
            }

            await authenticatedFetch(`${API_BASE_URL}/test/submit`, {
                method: 'POST',
                body: JSON.stringify(payload)
            });

            setSuccess('Test submitted successfully!');
            // Reset form
            setFormData({
                name: '',
                durationSeconds: '30s',
                ratePerSecond: 10,
                workerCount: 1,
                rateDistribution: 'shared',
                rateWeights: [],
                rampDuration: '',
                rampStartDelay: '',
                rampSteps: '',
                targets: '[{"method":"GET","url":"https://httpbin.org/get"}]',
                timeout: 30,
                redirects: 10,
                keepalive: true,
                http2: false,
                insecure: false,
                connections: 10000,
                vegetaPayloadJson: '{}'
            });
        } catch (err) {
            setError(err.message);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <div className="max-w-4xl mx-auto">
            <div className="mb-8">
                <h1 className="text-3xl font-bold text-gray-900">Create New Load Test</h1>
                <p className="text-gray-600 mt-2">Configure and launch a new distributed load test</p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-8">
                {/* Basic Configuration */}
                <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900 mb-4">Basic Configuration</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Test Name</label>
                            <input
                                type="text"
                                name="name"
                                value={formData.name}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                required
                                placeholder="My Load Test"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Duration</label>
                            <input
                                type="text"
                                name="durationSeconds"
                                value={formData.durationSeconds}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                required
                                placeholder="30s, 5m, 1h"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Rate (req/sec)</label>
                            <input
                                type="number"
                                name="ratePerSecond"
                                value={formData.ratePerSecond}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                required
                                min="1"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Worker Count</label>
                            <input
                                type="number"
                                name="workerCount"
                                value={formData.workerCount}
                                onChange={(e) => updateWorkerCount(parseInt(e.target.value))}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                required
                                min="1"
                            />
                        </div>
                    </div>
                </div>

                {/* Rate Distribution */}
                <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900 mb-4">Rate Distribution</h2>
                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Distribution Mode</label>
                            <select
                                name="rateDistribution"
                                value={formData.rateDistribution}
                                onChange={(e) => handleRateDistributionChange(e.target.value)}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            >
                                <option value="shared">Shared - Split rate across workers</option>
                                <option value="same">Same - Each worker uses full rate</option>
                                <option value="weighted">Weighted - Custom weights per worker</option>
                                <option value="ramped">Ramped - Gradually increase rate</option>
                                <option value="burst">Burst - High load on fewer workers</option>
                            </select>
                        </div>

                        {formData.rateDistribution === 'weighted' && (
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-2">Worker Weights</label>
                                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                                    {Array(formData.workerCount).fill(0).map((_, index) => (
                                        <div key={index}>
                                            <label className="block text-xs text-gray-600 mb-1">Worker {index + 1}</label>
                                            <input
                                                type="number"
                                                step="0.1"
                                                value={formData.rateWeights[index] || 1.0}
                                                onChange={(e) => handleWeightsChange(index, e.target.value)}
                                                className="w-full px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                                min="0.1"
                                            />
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {formData.rateDistribution === 'ramped' && (
                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-2">Ramp Duration</label>
                                    <input
                                        type="text"
                                        name="rampDuration"
                                        value={formData.rampDuration}
                                        onChange={handleInputChange}
                                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                        placeholder="30s (optional)"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-2">Start Delay</label>
                                    <input
                                        type="text"
                                        name="rampStartDelay"
                                        value={formData.rampStartDelay}
                                        onChange={handleInputChange}
                                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                        placeholder="5s (optional)"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-2">Steps</label>
                                    <input
                                        type="number"
                                        name="rampSteps"
                                        value={formData.rampSteps}
                                        onChange={handleInputChange}
                                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                        placeholder="5 (optional)"
                                        min="1"
                                    />
                                </div>
                            </div>
                        )}
                    </div>
                </div>

                {/* HTTP Configuration */}
                <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900 mb-4">HTTP Configuration</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Timeout (seconds)</label>
                            <input
                                type="number"
                                name="timeout"
                                value={formData.timeout}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                min="1"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Max Redirects</label>
                            <input
                                type="number"
                                name="redirects"
                                value={formData.redirects}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                min="0"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Max Connections</label>
                            <input
                                type="number"
                                name="connections"
                                value={formData.connections}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                min="1"
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
                        <label className="flex items-center space-x-3">
                            <input
                                type="checkbox"
                                name="keepalive"
                                checked={formData.keepalive}
                                onChange={handleInputChange}
                                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                            />
                            <span className="text-sm font-medium text-gray-700">Keep-Alive</span>
                        </label>

                        <label className="flex items-center space-x-3">
                            <input
                                type="checkbox"
                                name="http2"
                                checked={formData.http2}
                                onChange={handleInputChange}
                                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                            />
                            <span className="text-sm font-medium text-gray-700">HTTP/2</span>
                        </label>

                        <label className="flex items-center space-x-3">
                            <input
                                type="checkbox"
                                name="insecure"
                                checked={formData.insecure}
                                onChange={handleInputChange}
                                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                            />
                            <span className="text-sm font-medium text-gray-700">Skip TLS Verify</span>
                        </label>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Advanced Vegeta Options (JSON)</label>
                        <textarea
                            name="vegetaPayloadJson"
                            value={formData.vegetaPayloadJson}
                            onChange={handleInputChange}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            rows="3"
                            placeholder='{"headers": {"User-Agent": "MyApp/1.0"}}'
                        />
                        <p className="text-xs text-gray-600 mt-1">Additional Vegeta options in JSON format. Will merge with options above.</p>
                    </div>
                </div>

                {/* Targets */}
                <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900 mb-4">Test Targets</h2>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Targets (JSON)</label>
                        <textarea
                            name="targets"
                            value={formData.targets}
                            onChange={handleInputChange}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            rows="6"
                            required
                            placeholder='[{"method":"GET","url":"https://example.com/api/endpoint"}]'
                        />
                        <p className="text-xs text-gray-600 mt-1">JSON array of HTTP targets to test</p>
                    </div>
                </div>

                {/* Submit */}
                <div className="flex items-center justify-between">
                    {error && (
                        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg flex-1 mr-4">
                            {error}
                        </div>
                    )}

                    {success && (
                        <div className="bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded-lg flex-1 mr-4">
                            {success}
                        </div>
                    )}

                    <button
                        type="submit"
                        disabled={submitting}
                        className="flex items-center space-x-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-medium"
                    >
                        {submitting ? <LoadingSpinner size="small" /> : <Rocket className="w-5 h-5" />}
                        <span>{submitting ? 'Submitting...' : 'Launch Test'}</span>
                    </button>
                </div>
            </form>
        </div>
    );
};

// --- Main App Component ---
const App = () => {
    const { isLoggedIn } = React.useContext(AuthContext);
    const [currentPage, setCurrentPage] = useState('dashboard');

    // Handle auth errors
    useEffect(() => {
        const handleAuthError = () => {
            const { logout } = React.useContext(AuthContext);
            logout();
        };

        window.addEventListener('auth-error', handleAuthError);
        return () => window.removeEventListener('auth-error', handleAuthError);
    }, []);

    if (!isLoggedIn) {
        return <LoginPage />;
    }

    const renderCurrentPage = () => {
        switch (currentPage) {
            case 'dashboard':
                return <DashboardPage />;
            case 'new-test':
                return <NewTestPage />;
            case 'test-history':
                return <TestHistoryPage />;
            default:
                return <DashboardPage />;
        }
    };

    return (
        <div className="min-h-screen bg-gray-50">
            <Navbar currentPage={currentPage} setCurrentPage={setCurrentPage} />
            <main className="max-w-7xl mx-auto px-6 py-8">
                {renderCurrentPage()}
            </main>
        </div>
    );
};

// --- Root Render ---
const root = createRoot(document.getElementById('root'));
root.render(
    <AuthProvider>
        <App />
    </AuthProvider>
);
