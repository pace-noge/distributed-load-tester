import React, { useState, useEffect, useCallback } from 'react';
import {
    Users, CheckCircle, Clock, BarChart3, TrendingUp, Zap, Target
} from 'lucide-react';

import { LoadingSpinner, StatusBadge } from '../components/common/UIComponents.jsx';
import { useAuth } from '../contexts/AuthContext.jsx';
import { authenticatedFetch } from '../utils/api.js';
import { API_BASE_URL } from '../utils/constants.js';

export const DashboardPage = () => {
    const [dashboardData, setDashboardData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [connectionStatus, setConnectionStatus] = useState('connecting');
    const [usingWebSocket, setUsingWebSocket] = useState(false);
    const { getToken } = useAuth();

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
            <DashboardHeader connectionStatus={connectionStatus} usingWebSocket={usingWebSocket} />

            {error && <ErrorMessage message={error} />}

            {dashboardData && (
                <>
                    <WorkerStatusCards dashboardData={dashboardData} />
                    {dashboardData.active_tests && dashboardData.active_tests.length > 0 && (
                        <ActiveTestsSection tests={dashboardData.active_tests} />
                    )}
                    {dashboardData.worker_summaries && dashboardData.worker_summaries.length > 0 && (
                        <WorkerStatusTable workers={dashboardData.worker_summaries} />
                    )}
                </>
            )}
        </div>
    );
};

const DashboardHeader = ({ connectionStatus, usingWebSocket }) => (
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
);

const ErrorMessage = ({ message }) => (
    <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
        {message}
    </div>
);

const WorkerStatusCards = ({ dashboardData }) => (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatusCard
            icon={Users}
            value={dashboardData.total_workers}
            label="Total Workers"
            color="blue"
        />
        <StatusCard
            icon={CheckCircle}
            value={dashboardData.available_workers}
            label="Available"
            color="green"
        />
        <StatusCard
            icon={Clock}
            value={dashboardData.busy_workers}
            label="Busy"
            color="yellow"
        />
    </div>
);

const StatusCard = ({ icon: Icon, value, label, color }) => {
    const colorClasses = {
        blue: { bg: 'bg-blue-50', border: 'border-blue-200', iconBg: 'bg-blue-600', text: 'text-blue-900', subtext: 'text-blue-700' },
        green: { bg: 'bg-green-50', border: 'border-green-200', iconBg: 'bg-green-600', text: 'text-green-900', subtext: 'text-green-700' },
        yellow: { bg: 'bg-yellow-50', border: 'border-yellow-200', iconBg: 'bg-yellow-600', text: 'text-yellow-900', subtext: 'text-yellow-700' }
    };

    const classes = colorClasses[color];

    return (
        <div className={`${classes.bg} p-6 rounded-lg border ${classes.border}`}>
            <div className="flex items-center space-x-4">
                <div className={`p-3 ${classes.iconBg} rounded-lg`}>
                    <Icon className="w-6 h-6 text-white" />
                </div>
                <div>
                    <div className={`text-2xl font-bold ${classes.text}`}>
                        {value}
                    </div>
                    <div className={`text-sm ${classes.subtext}`}>{label}</div>
                </div>
            </div>
        </div>
    );
};

const ActiveTestsSection = ({ tests }) => (
    <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Active Tests</h2>
        <div className="space-y-4">
            {tests.map((test) => (
                <ActiveTestCard key={test.test_id} test={test} />
            ))}
        </div>
    </div>
);

const ActiveTestCard = ({ test }) => (
    <div className="border border-gray-200 rounded-lg p-4">
        <div className="flex items-center justify-between mb-3">
            <div>
                <h3 className="font-medium text-gray-900">{test.test_name}</h3>
                <p className="text-sm text-gray-600">{test.test_id}</p>
            </div>
            <StatusBadge status={test.status} />
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-3">
            <TestMetric value={test.assigned_workers} label="Assigned" />
            <TestMetric value={test.completed_workers} label="Completed" color="green" />
            <TestMetric value={test.total_requests_sent.toLocaleString()} label="Requests Sent" color="blue" />
            <TestMetric value={`${(test.progress * 100).toFixed(1)}%`} label="Progress" color="purple" />
        </div>

        <div className="w-full bg-gray-200 rounded-full h-2">
            <div
                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                style={{ width: `${test.progress * 100}%` }}
            ></div>
        </div>
    </div>
);

const TestMetric = ({ value, label, color = "gray" }) => {
    const colorClasses = {
        gray: 'text-gray-900',
        green: 'text-green-600',
        blue: 'text-blue-600',
        purple: 'text-purple-600'
    };

    return (
        <div className="text-center">
            <div className={`text-lg font-semibold ${colorClasses[color]}`}>
                {value}
            </div>
            <div className="text-xs text-gray-600">{label}</div>
        </div>
    );
};

const WorkerStatusTable = ({ workers }) => (
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
                    {workers.map((worker) => (
                        <WorkerRow key={worker.worker_id} worker={worker} />
                    ))}
                </tbody>
            </table>
        </div>
    </div>
);

const WorkerRow = ({ worker }) => (
    <tr>
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
);
