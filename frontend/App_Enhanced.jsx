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
