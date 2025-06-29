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
            const data = await authenticatedFetch(`${API_BASE_URL}/tests/history?page=${page}&limit=${pageSize}`);
            setTests(data.tests || []);
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
                                                {formatDate(test.created_at)}
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
