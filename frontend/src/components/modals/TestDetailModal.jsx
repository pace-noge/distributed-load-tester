import React, { useState, useEffect, useCallback } from 'react';
import {
    Play, XCircle, Target, Users, Clock, BarChart3,
    CheckCircle, AlertCircle
} from 'lucide-react';
import {
    BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
    LineChart, Line, PieChart, Pie, Cell
} from 'recharts';

import { LoadingSpinner, StatusBadge } from '../common/UIComponents.jsx';
import { formatDate, formatDuration, formatRateDistribution, getValueWithFallback } from '../../utils/formatters.js';
import { fetchTestDetail, replayTest } from '../../utils/api.js';
import { COLORS, MODAL_TABS } from '../../utils/constants.js';

export const TestDetailModal = ({ testId, isOpen, onClose }) => {
    const [testDetail, setTestDetail] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [activeTab, setActiveTab] = useState('overview');

    const handleFetchTestDetail = useCallback(async () => {
        if (!testId || !isOpen) return;

        setLoading(true);
        setError('');
        try {
            const data = await fetchTestDetail(testId);
            setTestDetail(data);
        } catch (err) {
            console.error('Error fetching test detail:', err);
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }, [testId, isOpen]);

    useEffect(() => {
        handleFetchTestDetail();
    }, [handleFetchTestDetail]);

    const handleReplay = async () => {
        try {
            await replayTest(testId, testDetail?.test?.name);
            alert('Test replayed successfully!');
            onClose();
        } catch (err) {
            alert(`Failed to replay test: ${err.message}`);
        }
    };

    if (!isOpen) return null;

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
                        {MODAL_TABS.map((tab) => (
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
                        <ModalContent
                            activeTab={activeTab}
                            testDetail={testDetail}
                            workerPerformanceData={workerPerformanceData}
                            statusCodeData={statusCodeData}
                        />
                    )}
                </div>
            </div>
        </div>
    );
};

const ModalContent = ({ activeTab, testDetail, workerPerformanceData, statusCodeData }) => {
    const test = testDetail.test;

    if (activeTab === 'overview') {
        return <OverviewTab test={test} />;
    }

    if (activeTab === 'results') {
        return <ResultsTab testDetail={testDetail} />;
    }

    if (activeTab === 'charts') {
        return <ChartsTab workerPerformanceData={workerPerformanceData} statusCodeData={statusCodeData} />;
    }

    return null;
};

const OverviewTab = ({ test }) => (
    <div className="space-y-6">
        {/* Test Info */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <InfoCard
                icon={Target}
                value={getValueWithFallback(test, ['ratePerSecond', 'rate_per_second'])}
                label="Req/sec"
                bgColor="bg-blue-50"
                textColor="text-blue-900"
                iconColor="text-blue-600"
            />
            <InfoCard
                icon={Users}
                value={getValueWithFallback(test, ['workerCount', 'worker_count'], 1)}
                label="Workers"
                bgColor="bg-green-50"
                textColor="text-green-900"
                iconColor="text-green-600"
            />
            <InfoCard
                icon={Clock}
                value={formatDuration(getValueWithFallback(test, ['durationSeconds', 'duration_seconds']))}
                label="Duration"
                bgColor="bg-yellow-50"
                textColor="text-yellow-900"
                iconColor="text-yellow-600"
            />
            <InfoCard
                icon={BarChart3}
                value={formatRateDistribution(getValueWithFallback(test, ['rateDistribution', 'rate_distribution']))}
                label="Distribution"
                bgColor="bg-purple-50"
                textColor="text-purple-900"
                iconColor="text-purple-600"
            />
        </div>

        {/* Status and Timeline */}
        <div className="bg-gray-50 p-6 rounded-lg">
            <h3 className="text-lg font-medium text-gray-900 mb-4">Test Status</h3>
            <div className="flex items-center justify-between">
                <StatusBadge status={test.status} />
                <div className="text-sm text-gray-600">
                    Created: {formatDate(getValueWithFallback(test, ['createdAt', 'created_at', 'created']))}
                </div>
            </div>

            {test.assigned_workers_ids?.length > 0 && (
                <div className="mt-4">
                    <div className="text-sm font-medium text-gray-700 mb-2">Worker Status:</div>
                    <div className="flex flex-wrap gap-2">
                        {test.assigned_workers_ids.map((workerId) => (
                            <span key={workerId} className="px-2 py-1 bg-blue-100 text-blue-800 rounded text-xs">
                                {workerId}
                            </span>
                        ))}
                    </div>
                </div>
            )}
        </div>
    </div>
);

const ResultsTab = ({ testDetail }) => (
    <div className="space-y-6">
        {testDetail.aggregated_result ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <InfoCard
                    value={testDetail.aggregated_result.total_requests.toLocaleString()}
                    label="Total Requests"
                    bgColor="bg-blue-50"
                    textColor="text-blue-900"
                />
                <InfoCard
                    value={testDetail.aggregated_result.successful_requests.toLocaleString()}
                    label="Successful"
                    bgColor="bg-green-50"
                    textColor="text-green-900"
                />
                <InfoCard
                    value={testDetail.aggregated_result.failed_requests.toLocaleString()}
                    label="Failed"
                    bgColor="bg-red-50"
                    textColor="text-red-900"
                />
                <InfoCard
                    value={`${testDetail.aggregated_result.avg_latency_ms.toFixed(2)}ms`}
                    label="Avg Latency"
                    bgColor="bg-yellow-50"
                    textColor="text-yellow-900"
                />
            </div>
        ) : (
            <div className="text-gray-500 text-center py-8">
                No aggregated results available yet.
            </div>
        )}

        {/* Individual Worker Results */}
        {testDetail.results?.length > 0 && <WorkerResultsTable results={testDetail.results} />}
    </div>
);

const ChartsTab = ({ workerPerformanceData, statusCodeData }) => (
    <div className="space-y-8">
        {workerPerformanceData.length > 0 && (
            <>
                {/* Worker Performance Chart */}
                <ChartContainer title="Worker Performance">
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
                </ChartContainer>

                {/* Latency Chart */}
                <ChartContainer title="Latency Distribution">
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
                </ChartContainer>
            </>
        )}

        {/* Status Code Distribution */}
        {statusCodeData.length > 0 && (
            <ChartContainer title="Status Code Distribution">
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
            </ChartContainer>
        )}

        {workerPerformanceData.length === 0 && statusCodeData.length === 0 && (
            <div className="text-gray-500 text-center py-12">
                No chart data available. Results may still be processing.
            </div>
        )}
    </div>
);

// Helper Components
const InfoCard = ({ icon: Icon, value, label, bgColor, textColor, iconColor }) => (
    <div className={`${bgColor} p-4 rounded-lg`}>
        <div className="flex items-center space-x-3">
            {Icon && <Icon className={`w-8 h-8 ${iconColor}`} />}
            <div>
                <div className={`text-2xl font-bold ${textColor}`}>
                    {value}
                </div>
                <div className={`text-sm ${iconColor}`}>{label}</div>
            </div>
        </div>
    </div>
);

const WorkerResultsTable = ({ results }) => (
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
                {results.map((result, index) => (
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
);

const ChartContainer = ({ title, children }) => (
    <div className="bg-white p-6 rounded-lg border border-gray-200">
        <h3 className="text-lg font-medium text-gray-900 mb-4">{title}</h3>
        {children}
    </div>
);
