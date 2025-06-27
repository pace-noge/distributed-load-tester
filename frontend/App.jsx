import React, { useState, useEffect, useCallback } from 'react';
import { createRoot } from 'react-dom/client';
import { ChevronRight, LayoutDashboard, Rocket, List, History, User, LogOut } from 'lucide-react'; // Icons

// Ensure Tailwind CSS is loaded, assumed to be available in the environment
// <script src="https://cdn.tailwindcss.com"></script>

const API_BASE_URL = 'http://localhost:8080/api'; // Update if your Master HTTP port is different
const AUTH_URL = 'http://localhost:8080/auth/login';

// --- Auth Context for managing authentication state ---
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

// --- API Utility with Auth Token ---
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
            // Token expired or invalid, trigger logout
            const event = new CustomEvent('auth-error');
            window.dispatchEvent(event);
        }
        const errorData = await response.json().catch(() => ({ message: response.statusText }));
        throw new Error(errorData.message || `API Error: ${response.status}`);
    }
    return response.json();
};


// --- Login Page Component ---
const LoginPage = () => {
    const [username, setUsername] = useState('admin'); // Default for convenience
    const [password, setPassword] = useState('password'); // Default for convenience
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
                headers: {
                    'Content-Type': 'application/json',
                },
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
        <div className="min-h-screen flex items-center justify-center bg-gray-100 p-4">
            <div className="bg-white p-8 rounded-lg shadow-xl w-full max-w-md">
                <h2 className="text-3xl font-bold text-center text-gray-800 mb-6">Login to Load Tester</h2>
                <form onSubmit={handleLogin} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1" htmlFor="username">Username</label>
                        <input
                            type="text"
                            id="username"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                            required
                            disabled={loading}
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1" htmlFor="password">Password</label>
                        <input
                            type="password"
                            id="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                            required
                            disabled={loading}
                        />
                    </div>
                    {error && (
                        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" role="alert">
                            <span className="block sm:inline">{error}</span>
                        </div>
                    )}
                    <button
                        type="submit"
                        className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 flex items-center justify-center"
                        disabled={loading}
                    >
                        {loading && <span className="mr-2 animate-spin rounded-full h-4 w-4 border-b-2 border-white"></span>}
                        {loading ? 'Logging in...' : 'Login'}
                    </button>
                </form>
            </div>
        </div>
    );
};

// --- Navbar Component ---
const Navbar = ({ currentPage, setCurrentPage }) => {
    const { logout } = React.useContext(AuthContext);

    const navItems = [
        { name: 'Dashboard', icon: LayoutDashboard, page: 'dashboard' },
        { name: 'Submit Test', icon: Rocket, page: 'submit' },
        { name: 'Test History', icon: List, page: 'history' },
    ];

    return (
        <nav className="bg-gray-800 p-4 text-white shadow-lg">
            <div className="container mx-auto flex justify-between items-center">
                <div className="text-2xl font-bold">Load Tester</div>
                <ul className="flex space-x-6">
                    {navItems.map(item => (
                        <li key={item.page}>
                            <button
                                onClick={() => setCurrentPage(item.page)}
                                className={`flex items-center space-x-2 p-2 rounded-md transition-colors duration-200 
                                    ${currentPage === item.page ? 'bg-blue-600' : 'hover:bg-gray-700'}`}
                            >
                                <item.icon size={18} />
                                <span className="hidden md:inline">{item.name}</span>
                            </button>
                        </li>
                    ))}
                    <li>
                        <button
                            onClick={logout}
                            className="flex items-center space-x-2 p-2 rounded-md bg-red-600 hover:bg-red-700 transition-colors duration-200"
                        >
                            <LogOut size={18} />
                            <span className="hidden md:inline">Logout</span>
                        </button>
                    </li>
                </ul>
            </div>
        </nav>
    );
};


// --- Dashboard Component ---
const DashboardPage = () => {
    const [dashboardData, setDashboardData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [connectionStatus, setConnectionStatus] = useState('connecting');
    const [usingWebSocket, setUsingWebSocket] = useState(true);
    const { getToken } = React.useContext(AuthContext);

    // Fallback HTTP polling function
    const fetchDashboardData = useCallback(async () => {
        try {
            console.log('Fetching dashboard data via HTTP...');
            const data = await authenticatedFetch(`${API_BASE_URL}/dashboard`);
            setDashboardData(data);
            setError('');
            setLoading(false);
        } catch (err) {
            console.error("Failed to fetch dashboard data:", err);
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

        console.log('Starting WebSocket connection...');
        
        // Create WebSocket connection
        const wsUrl = `ws://localhost:8080/ws?token=${encodeURIComponent(token)}`;
        console.log('WebSocket URL:', wsUrl);
        
        const ws = new WebSocket(wsUrl);
        let wsConnected = false;

        // Fallback timer - if WebSocket doesn't connect in 5 seconds, use HTTP polling
        const fallbackTimer = setTimeout(() => {
            if (!wsConnected) {
                console.log('WebSocket connection timeout, falling back to HTTP polling');
                setUsingWebSocket(false);
                setConnectionStatus('http_polling');
                fetchDashboardData();
                
                // Start HTTP polling
                const interval = setInterval(fetchDashboardData, 5000);
                return () => clearInterval(interval);
            }
        }, 5000);

        ws.onopen = () => {
            console.log('WebSocket connected successfully');
            wsConnected = true;
            clearTimeout(fallbackTimer);
            setConnectionStatus('connected');
            setUsingWebSocket(true);
            setError('');
        };

        ws.onmessage = (event) => {
            console.log('WebSocket message received:', event.data);
            try {
                const message = JSON.parse(event.data);
                console.log('Parsed message:', message);
                if (message.type === 'dashboard_update') {
                    console.log('Setting dashboard data:', message.data);
                    setDashboardData(message.data);
                    setLoading(false);
                    setError('');
                }
            } catch (err) {
                console.error('Error parsing WebSocket message:', err);
                setError('Error parsing server message');
            }
        };

        ws.onclose = (event) => {
            console.log('WebSocket closed:', event.code, event.reason);
            setConnectionStatus('disconnected');
            if (event.code !== 1000) { // Not a normal closure
                setError(`Connection lost (code: ${event.code}). Falling back to HTTP polling...`);
                setUsingWebSocket(false);
                // Start HTTP polling as fallback
                fetchDashboardData();
                const interval = setInterval(fetchDashboardData, 5000);
                setTimeout(() => clearInterval(interval), 60000); // Clear after 1 minute
            }
        };

        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            setConnectionStatus('error');
            setError('WebSocket connection failed. Falling back to HTTP polling...');
            setUsingWebSocket(false);
            clearTimeout(fallbackTimer);
            // Fallback to HTTP polling
            fetchDashboardData();
            const interval = setInterval(fetchDashboardData, 5000);
            setTimeout(() => clearInterval(interval), 60000); // Clear after 1 minute
        };

        // Cleanup on unmount
        return () => {
            console.log('Cleaning up WebSocket connection...');
            clearTimeout(fallbackTimer);
            if (ws.readyState === WebSocket.OPEN) {
                ws.close(1000, 'Component unmounting');
            }
        };
    }, [getToken, fetchDashboardData]);

    if (loading && !dashboardData) {
        return (
            <div className="flex justify-center items-center h-48">
                <span className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></span>
                <p className="ml-3 text-gray-700">Loading Dashboard...</p>
            </div>
        );
    }

    if (error) {
        return (
            <div className="container mx-auto p-4 mt-8">
                <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" role="alert">
                    <p className="font-bold">Error!</p>
                    <p>{error}</p>
                    {connectionStatus === 'disconnected' && (
                        <p className="text-sm mt-2">Status: Disconnected</p>
                    )}
                </div>
            </div>
        );
    }

    if (!dashboardData) {
        return <div className="container mx-auto p-4 mt-8 text-center text-gray-600">No dashboard data available.</div>;
    }

    return (
        <div className="container mx-auto p-4 py-8">
            <div className="flex justify-between items-center mb-8">
                <h1 className="text-3xl font-bold text-gray-800">Dashboard</h1>
                <div className="flex items-center">
                    <div className={`w-3 h-3 rounded-full mr-2 ${
                        connectionStatus === 'connected' ? 'bg-green-500' : 
                        connectionStatus === 'connecting' ? 'bg-yellow-500' : 
                        connectionStatus === 'http_polling' ? 'bg-blue-500' : 'bg-red-500'
                    }`}></div>
                    <span className="text-sm text-gray-600">
                        {connectionStatus === 'connected' && usingWebSocket && 'WebSocket Connected'}
                        {connectionStatus === 'connecting' && 'Connecting...'}
                        {connectionStatus === 'http_polling' && 'HTTP Polling'}
                        {connectionStatus === 'disconnected' && 'Disconnected'}
                        {connectionStatus === 'error' && 'Connection Error'}
                    </span>
                </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
                <div className="bg-white p-6 rounded-lg shadow-md border-b-4 border-blue-500">
                    <p className="text-gray-500 text-sm font-medium">Total Workers</p>
                    <p className="text-3xl font-bold text-gray-900 mt-1">{dashboardData.total_workers}</p>
                </div>
                <div className="bg-white p-6 rounded-lg shadow-md border-b-4 border-green-500">
                    <p className="text-gray-500 text-sm font-medium">Available Workers</p>
                    <p className="text-3xl font-bold text-gray-900 mt-1">{dashboardData.available_workers}</p>
                </div>
                <div className="bg-white p-6 rounded-lg shadow-md border-b-4 border-yellow-500">
                    <p className="text-gray-500 text-sm font-medium">Busy Workers</p>
                    <p className="text-3xl font-bold text-gray-900 mt-1">{dashboardData.busy_workers}</p>
                </div>
            </div>

            <h2 className="text-2xl font-bold text-gray-800 mb-6">Active Tests</h2>
            {dashboardData.active_tests && dashboardData.active_tests.length > 0 ? (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    {dashboardData.active_tests.map(test => (
                        <div key={test.test_id} className="bg-white p-6 rounded-lg shadow-md border-t-4 border-purple-500">
                            <h3 className="text-xl font-semibold text-gray-800 mb-2">{test.test_name}</h3>
                            <p className="text-sm text-gray-600 mb-2">ID: <span className="font-mono text-xs">{test.test_id}</span></p>
                            <p className="text-sm text-gray-600 mb-2">Status: <span className={`font-medium ${test.status === 'RUNNING' ? 'text-blue-600' : test.status === 'COMPLETED' ? 'text-green-600' : 'text-orange-600'}`}>{test.status}</span></p>
                            <div className="w-full bg-gray-200 rounded-full h-2.5 mb-2">
                                <div className="bg-blue-600 h-2.5 rounded-full" style={{ width: `${(test.total_requests_completed / test.total_requests_sent) * 100 || 0}%` }}></div>
                            </div>
                            <p className="text-xs text-gray-500 text-right">{((test.total_requests_completed / test.total_requests_sent) * 100 || 0).toFixed(1)}% Completed</p>
                            <div className="grid grid-cols-2 gap-2 text-sm mt-3">
                                <div><span className="font-medium">Assigned Workers:</span> {test.assigned_workers}</div>
                                <div><span className="font-medium">Completed Workers:</span> {test.completed_workers}</div>
                                <div><span className="font-medium">Failed Workers:</span> {test.failed_workers}</div>
                                <div><span className="font-medium">Requests Sent:</span> {test.total_requests_sent}</div>
                                <div><span className="font-medium">Requests Completed:</span> {test.total_requests_completed}</div>
                            </div>
                        </div>
                    ))}
                </div>
            ) : (
                <p className="text-gray-600">No tests currently running or pending.</p>
            )}

            <h2 className="text-2xl font-bold text-gray-800 my-6">Worker Status Summary</h2>
            {dashboardData.worker_summaries && dashboardData.worker_summaries.length > 0 ? (
                <div className="bg-white rounded-lg shadow-md overflow-hidden">
                    <table className="min-w-full divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                            <tr>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Worker ID</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Current Test</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Progress</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Message</th>
                            </tr>
                        </thead>
                        <tbody className="bg-white divide-y divide-gray-200">
                            {dashboardData.worker_summaries.map(worker => (
                                <tr key={worker.worker_id}>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{worker.worker_id}</td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                                        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                                            ${worker.status_type === 'READY' ? 'bg-green-100 text-green-800' :
                                              worker.status_type === 'BUSY' ? 'bg-blue-100 text-blue-800' :
                                              worker.status_type === 'FINISHING' ? 'bg-yellow-100 text-yellow-800' :
                                              'bg-red-100 text-red-800'}`}>
                                            {worker.status_type}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {worker.current_test_id || '-'}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {worker.total_requests > 0 ? `${worker.completed_requests}/${worker.total_requests}` : '-'}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {worker.status_message}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            ) : (
                <p className="text-gray-600">No workers registered.</p>
            )}
        </div>
    );
};

// --- Test Submission Page Component ---
const TestSubmissionPage = () => {
    const [testName, setTestName] = useState('');
    const [duration, setDuration] = useState('30s');
    const [rate, setRate] = useState(10);
    const [targets, setTargets] = useState(''); // Raw targets data
    const [vegetaPayload, setVegetaPayload] = useState('{}'); // JSON string
    const [message, setMessage] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const handleSubmit = async (e) => {
        e.preventDefault();
        setMessage('');
        setError('');
        setLoading(true);

        try {
            // Base64 encode targets
            const encodedTargets = btoa(targets);

            const testData = {
                name: testName,
                duration_seconds: duration,
                rate_per_second: parseInt(rate),
                targets_base64: encodedTargets,
                vegeta_payload_json: vegetaPayload,
            };

            // Call Master API to submit test
            const response = await authenticatedFetch(`${API_BASE_URL}/test/submit`, {
                method: 'POST',
                body: JSON.stringify(testData),
            });

            setMessage(`Test submitted successfully! Test ID: ${response.testId}`);
            setTestName('');
            setDuration('30s');
            setRate(10);
            setTargets('');
            setVegetaPayload('{}');
        } catch (err) {
            console.error("Failed to submit test:", err);
            setError(`Failed to submit test: ${err.message}`);
        } finally {
            setLoading(false);
        }
    };

    // Correctly format the JSON placeholder string using JSON.stringify
    const exampleTargetsPlaceholder = JSON.stringify([
        { method: "GET", url: "http://localhost:8080/api/dashboard" }
    ], null, 2);

    return (
        <div className="container mx-auto p-4 py-8">
            <h1 className="text-3xl font-bold text-gray-800 mb-8">Submit New Load Test</h1>
            <form onSubmit={handleSubmit} className="bg-white p-8 rounded-lg shadow-md space-y-6">
                <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1" htmlFor="testName">Test Name</label>
                    <input
                        type="text"
                        id="testName"
                        value={testName}
                        onChange={(e) => setTestName(e.target.value)}
                        className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                        required
                        disabled={loading}
                    />
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1" htmlFor="duration">Duration (e.g., 10s, 1m)</label>
                        <input
                            type="text"
                            id="duration"
                            value={duration}
                            onChange={(e) => setDuration(e.target.value)}
                            className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                            required
                            disabled={loading}
                        />
                    </div>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1" htmlFor="rate">Rate (requests per second)</label>
                        <input
                            type="number"
                            id="rate"
                            value={rate}
                            onChange={(e) => setRate(e.target.value)}
                            className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                            required
                            min="1"
                            disabled={loading}
                        />
                    </div>
                </div>
                <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1" htmlFor="targets">Targets (JSON or plain text, e.g., [{`"method":"GET","url":"http://example.com"`}] or GET http://example.com)</label>
                    <textarea
                        id="targets"
                        value={targets}
                        onChange={(e) => setTargets(e.target.value)}
                        rows="6"
                        className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 font-mono"
                        placeholder={exampleTargetsPlaceholder} // Use the correctly formatted placeholder string
                        required
                        disabled={loading}
                    ></textarea>
                </div>
                <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1" htmlFor="vegetaPayload">Vegeta Attack Options (JSON, optional)</label>
                    <textarea
                        id="vegetaPayload"
                        value={vegetaPayload}
                        onChange={(e) => setVegetaPayload(e.target.value)}
                        rows="4"
                        className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 font-mono"
                        placeholder='{"timeout": 5}'
                        disabled={loading}
                    ></textarea>
                </div>

                {message && (
                    <div className="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded relative" role="alert">
                        <span className="block sm:inline">{message}</span>
                    </div>
                )}
                {error && (
                    <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" role="alert">
                        <span className="block sm:inline">{error}</span>
                    </div>
                )}

                <button
                    type="submit"
                    className="w-full bg-blue-600 text-white py-3 px-6 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 flex items-center justify-center text-lg font-semibold"
                    disabled={loading}
                >
                    {loading && <span className="mr-2 animate-spin rounded-full h-5 w-5 border-b-2 border-white"></span>}
                    {loading ? 'Submitting...' : 'Submit Test'}
                </button>
            </form>
        </div>
    );
};

// --- Test History Page Component ---
const TestHistoryPage = () => {
    const [tests, setTests] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [selectedTestId, setSelectedTestId] = useState(null);
    const [selectedTestResults, setSelectedTestResults] = useState(null);
    const [selectedTestAggregated, setSelectedTestAggregated] = useState(null);
    const [resultsLoading, setResultsLoading] = useState(false);
    const [resultsError, setResultsError] = useState('');

    const fetchTests = useCallback(async () => {
        try {
            setLoading(true);
            const data = await authenticatedFetch(`${API_BASE_URL}/tests`);
            setTests(data);
            setError('');
        } catch (err) {
            console.error("Failed to fetch test history:", err);
            setError(`Failed to load test history: ${err.message}`);
        } finally {
            setLoading(false);
        }
    }, []);

    const fetchTestResults = useCallback(async (testId) => {
        setResultsLoading(true);
        setResultsError('');
        setSelectedTestResults(null);
        setSelectedTestAggregated(null);
        try {
            // Fetch raw results first
            const rawResults = await authenticatedFetch(`${API_BASE_URL}/tests/${testId}/results`);
            setSelectedTestResults(rawResults);

            // Try to fetch aggregated results, but handle gracefully if not ready
            try {
                const aggregatedResult = await authenticatedFetch(`${API_BASE_URL}/tests/${testId}/aggregated-result`);
                setSelectedTestAggregated(aggregatedResult);
            } catch (aggregateErr) {
                if (aggregateErr.message.includes('404') || aggregateErr.message.includes('not found')) {
                    // Aggregated results not ready yet - this is expected for recent tests
                    console.log(`Aggregated results not ready yet for test ${testId}:`, aggregateErr.message);
                    setSelectedTestAggregated('processing'); // Special marker to show "processing" state
                } else {
                    throw aggregateErr; // Re-throw other errors
                }
            }

        } catch (err) {
            console.error(`Failed to fetch results for ${testId}:`, err);
            setResultsError(`Failed to load results: ${err.message}`);
        } finally {
            setResultsLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchTests();
        // No auto-refresh for history, as it's static past data
    }, [fetchTests]);

    const handleViewResults = (testId) => {
        setSelectedTestId(testId);
        fetchTestResults(testId);
    };

    if (loading) {
        return (
            <div className="flex justify-center items-center h-48">
                <span className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></span>
                <p className="ml-3 text-gray-700">Loading Test History...</p>
            </div>
        );
    }

    if (error) {
        return (
            <div className="container mx-auto p-4 mt-8">
                <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative" role="alert">
                    <p className="font-bold">Error!</p>
                    <p>{error}</p>
                </div>
            </div>
        );
    }

    return (
        <div className="container mx-auto p-4 py-8">
            <h1 className="text-3xl font-bold text-gray-800 mb-8">Test History</h1>

            {tests.length === 0 ? (
                <p className="text-gray-600">No tests have been submitted yet.</p>
            ) : (
                <div className="bg-white rounded-lg shadow-md overflow-hidden mb-8">
                    <table className="min-w-full divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                            <tr>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Test Name</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Submitted At</th>
                                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="bg-white divide-y divide-gray-200">
                            {tests.map(test => (
                                <tr key={test.id}>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{test.name}</td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                                        <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                                            ${test.status === 'COMPLETED' ? 'bg-green-100 text-green-800' :
                                              test.status === 'RUNNING' ? 'bg-blue-100 text-blue-800' :
                                              test.status === 'PENDING' ? 'bg-yellow-100 text-yellow-800' :
                                              'bg-red-100 text-red-800'}`}>
                                            {test.status}
                                        </span>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {new Date(test.createdAt).toLocaleString()}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                                        <button
                                            onClick={() => handleViewResults(test.id)}
                                            className="text-blue-600 hover:text-blue-900 flex items-center"
                                        >
                                            View Results <ChevronRight size={16} className="ml-1" />
                                        </button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {selectedTestId && (
                <div className="mt-10 bg-white p-8 rounded-lg shadow-md border-t-4 border-indigo-500">
                    <h2 className="text-2xl font-bold text-gray-800 mb-6">Results for Test: <span className="font-mono text-xl">{selectedTestId}</span></h2>

                    {resultsLoading && (
                        <div className="flex justify-center items-center h-24">
                            <span className="animate-spin rounded-full h-6 w-6 border-b-2 border-gray-900"></span>
                            <p className="ml-3 text-gray-700">Loading results...</p>
                        </div>
                    )}

                    {resultsError && (
                        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative mb-4" role="alert">
                            <p className="font-bold">Error loading results:</p>
                            <p>{resultsError}</p>
                        </div>
                    )}

                    {selectedTestAggregated === 'processing' && (
                        <div className="mb-8">
                            <h3 className="text-xl font-semibold text-gray-800 mb-4">Aggregated Result</h3>
                            <div className="p-6 bg-yellow-50 border border-yellow-200 rounded-lg">
                                <div className="flex items-center">
                                    <span className="animate-spin rounded-full h-5 w-5 border-b-2 border-yellow-600 mr-3"></span>
                                    <p className="text-yellow-800">
                                        <span className="font-medium">Processing results...</span>
                                        <br />
                                        <span className="text-sm">Aggregated results are being calculated. Please check back in a moment.</span>
                                    </p>
                                </div>
                            </div>
                        </div>
                    )}

                    {selectedTestAggregated && selectedTestAggregated !== 'processing' && (
                        <div className="mb-8">
                            <h3 className="text-xl font-semibold text-gray-800 mb-4">Aggregated Result</h3>
                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 text-sm">
                                <div className="p-3 bg-gray-50 rounded-md"><span className="font-medium">Total Requests:</span> {selectedTestAggregated.total_requests}</div>
                                <div className="p-3 bg-gray-50 rounded-md"><span className="font-medium">Successful:</span> {selectedTestAggregated.successful_requests}</div>
                                <div className="p-3 bg-gray-50 rounded-md"><span className="font-medium">Failed:</span> {selectedTestAggregated.failed_requests}</div>
                                <div className="p-3 bg-gray-50 rounded-md"><span className="font-medium">Avg Latency:</span> {selectedTestAggregated.avg_latency_ms ? selectedTestAggregated.avg_latency_ms.toFixed(2) : 0} ms</div>
                                <div className="p-3 bg-gray-50 rounded-md"><span className="font-medium">P95 Latency:</span> {selectedTestAggregated.p95_latency_ms ? selectedTestAggregated.p95_latency_ms.toFixed(2) : 0} ms</div>
                                <div className="p-3 bg-gray-50 rounded-md"><span className="font-medium">Duration:</span> {selectedTestAggregated.duration_ms ? selectedTestAggregated.duration_ms / 1000 : 0} s</div>
                                <div className="p-3 bg-gray-50 rounded-md col-span-full"><span className="font-medium">Overall Status:</span> {selectedTestAggregated.overall_status}</div>
                                {selectedTestAggregated.error_rates && Object.keys(selectedTestAggregated.error_rates).length > 0 && (
                                    <div className="p-3 bg-red-50 rounded-md col-span-full">
                                        <span className="font-medium">Error Rates:</span>
                                        <pre className="text-xs font-mono bg-red-100 p-2 rounded mt-1 overflow-auto">{JSON.stringify(selectedTestAggregated.error_rates, null, 2)}</pre>
                                    </div>
                                )}
                            </div>
                        </div>
                    )}

                    {selectedTestResults && selectedTestResults.length > 0 && (
                        <div>
                            <h3 className="text-xl font-semibold text-gray-800 mb-4">Raw Worker Results</h3>
                            <div className="space-y-4">
                                {selectedTestResults.map(result => (
                                    <div key={result.id} className="bg-gray-50 p-4 rounded-lg border border-gray-200">
                                        <p className="text-sm font-semibold text-gray-800 mb-2">Worker ID: <span className="font-mono text-xs">{result.workerId}</span></p>
                                        <p className="text-xs text-gray-600 mb-1"><span className="font-medium">Timestamp:</span> {new Date(result.timestamp).toLocaleString()}</p>
                                        <p className="text-xs text-gray-600 mb-1"><span className="font-medium">Total Requests:</span> {result.totalRequests}</p>
                                        <p className="text-xs text-gray-600 mb-1"><span className="font-medium">Completed Requests:</span> {result.completedRequests}</p>
                                        <p className="text-xs text-gray-600 mb-1"><span className="font-medium">Success Rate:</span> {(result.successRate * 100).toFixed(2)}%</p>
                                        <p className="text-xs text-gray-600 mb-1"><span className="font-medium">Avg Latency:</span> {result.averageLatencyMs.toFixed(2)} ms</p>
                                        <p className="text-xs text-gray-600 mb-1"><span className="font-medium">P95 Latency:</span> {result.p95LatencyMs.toFixed(2)} ms</p>
                                        <details className="mt-2 text-sm text-gray-700">
                                            <summary className="cursor-pointer font-medium hover:text-blue-600">Raw Metric Output</summary>
                                            <pre className="mt-2 p-2 bg-gray-100 rounded-md text-xs font-mono overflow-auto max-h-48">
                                                {(() => {
                                                    try {
                                                        // Try to parse as JSON first
                                                        if (typeof result.metric === 'string') {
                                                            const parsed = JSON.parse(result.metric);
                                                            return JSON.stringify(parsed, null, 2);
                                                        } else if (result.metric && typeof result.metric === 'object') {
                                                            // Already an object
                                                            return JSON.stringify(result.metric, null, 2);
                                                        } else {
                                                            return 'No metric data available';
                                                        }
                                                    } catch (error) {
                                                        // If JSON parsing fails, try base64 decoding
                                                        try {
                                                            const decoded = atob(result.metric);
                                                            const parsed = JSON.parse(decoded);
                                                            return JSON.stringify(parsed, null, 2);
                                                        } catch (decodeError) {
                                                            // If all else fails, show raw data
                                                            return `Raw data: ${result.metric}`;
                                                        }
                                                    }
                                                })()}
                                            </pre>
                                        </details>
                                        <details className="mt-2 text-sm text-gray-700">
                                            <summary className="cursor-pointer font-medium hover:text-blue-600">Status Codes</summary>
                                            <pre className="mt-2 p-2 bg-gray-100 rounded-md text-xs font-mono overflow-auto max-h-48">
                                                {JSON.stringify(result.statusCodes, null, 2)}
                                            </pre>
                                        </details>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
};

// --- Main App Component ---
// --- Error Boundary Component ---
class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props);
        this.state = { hasError: false, error: null };
    }

    static getDerivedStateFromError(error) {
        return { hasError: true, error };
    }

    componentDidCatch(error, errorInfo) {
        console.error('Error caught by boundary:', error, errorInfo);
    }

    render() {
        if (this.state.hasError) {
            return (
                <div style={{ padding: '20px', color: 'red', fontFamily: 'Arial, sans-serif' }}>
                    <h2>Something went wrong.</h2>
                    <details style={{ marginTop: '10px' }}>
                        <summary>Error details</summary>
                        <pre style={{ marginTop: '10px', backgroundColor: '#f5f5f5', padding: '10px' }}>
                            {this.state.error?.toString()}
                        </pre>
                    </details>
                </div>
            );
        }

        return this.props.children;
    }
}

const App = () => {
    const [currentPage, setCurrentPage] = useState('dashboard'); // Default page
    const { isLoggedIn, logout } = React.useContext(AuthContext);

    useEffect(() => {
        const handleAuthError = () => {
            logout(); // Force logout on auth errors (e.g., expired token)
        };
        window.addEventListener('auth-error', handleAuthError);
        return () => window.removeEventListener('auth-error', handleAuthError);
    }, [logout]);


    let PageComponent;
    switch (currentPage) {
        case 'dashboard':
            PageComponent = DashboardPage;
            break;
        case 'submit':
            PageComponent = TestSubmissionPage;
            break;
        case 'history':
            PageComponent = TestHistoryPage;
            break;
        default:
            PageComponent = DashboardPage; // Fallback
    }

    if (!isLoggedIn) {
        return <LoginPage />;
    }

    return (
        <div className="min-h-screen bg-gray-50 font-sans text-gray-900 antialiased">
            <Navbar currentPage={currentPage} setCurrentPage={setCurrentPage} />
            <main>
                <PageComponent />
            </main>
        </div>
    );
};

// --- Render the App ---
// Ensure the root element exists in your HTML: <div id="root"></div>
const container = document.getElementById('root');
if (!container) {
    console.error('Root element not found!');
    document.body.innerHTML = '<div>Error: Root element not found</div>';
} else {
    const root = createRoot(container);
    try {
        root.render(
            <ErrorBoundary>
                <AuthProvider>
                    <App />
                </AuthProvider>
            </ErrorBoundary>
        );
        console.log('App rendered successfully');
    } catch (error) {
        console.error('Error rendering app:', error);
        container.innerHTML = `<div style="padding: 20px; color: red;">Error rendering app: ${error.message}</div>`;
    }
}

// Basic HTML for local testing (save as index.html)
/*
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Distributed Load Tester</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
    </style>
</head>
<body>
    <div id="root"></div>
    <script type="module" src="./App.jsx"></script>
</body>
</html>
*/
