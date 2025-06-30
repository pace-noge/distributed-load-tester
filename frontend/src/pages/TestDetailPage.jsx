import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
    ArrowLeft, Play, Target, Users, Clock, BarChart3
} from 'lucide-react';
import {
    BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
    LineChart, Line, PieChart, Pie, Cell
} from 'recharts';

import { LoadingSpinner, StatusBadge } from '../components/common/UIComponents.jsx';
import { formatDate, formatDuration, formatRateDistribution, getValueWithFallback } from '../utils/formatters.js';
import { fetchTestDetail, replayTest } from '../utils/api.js';
import { COLORS, MODAL_TABS } from '../utils/constants.js';

// Helper function to decode base64 worker metrics (moved from modal)
const decodeWorkerMetrics = (result) => {
    try {
        let decodedMetrics = null;

        const base64Fields = [
            'metric',
            'vegeta_metrics_base64',
            'metrics_base64',
            'result_base64',
            'vegeta_result_base64',
            'attack_result_base64'
        ];

        for (const field of base64Fields) {
            if (result[field]) {
                try {
                    decodedMetrics = JSON.parse(atob(result[field]));
                    break;
                } catch (e) {
                    continue;
                }
            }
        }

        if (decodedMetrics) {
            // Handle Vegeta-style metrics
            if (decodedMetrics.latencies) {
                const totalRequests = decodedMetrics.requests || 0;
                const errorCount = decodedMetrics.errors?.length || 0;

                let successRate = 0;
                if (typeof decodedMetrics.success === 'number') {
                    successRate = decodedMetrics.success;
                } else if (totalRequests > 0) {
                    successRate = (totalRequests - errorCount) / totalRequests;
                }

                return {
                    ...result,
                    total_requests: totalRequests,
                    completed_requests: totalRequests,
                    success_rate: successRate,
                    average_latency_ms: decodedMetrics.latencies?.mean ? decodedMetrics.latencies.mean / 1000000 : 0,
                    p95_latency_ms: decodedMetrics.latencies?.['95th'] ? decodedMetrics.latencies['95th'] / 1000000 : 0,
                    p99_latency_ms: decodedMetrics.latencies?.['99th'] ? decodedMetrics.latencies['99th'] / 1000000 : 0,
                    p50_latency_ms: decodedMetrics.latencies?.['50th'] ? decodedMetrics.latencies['50th'] / 1000000 : 0,
                    p90_latency_ms: decodedMetrics.latencies?.['90th'] ? decodedMetrics.latencies['90th'] / 1000000 : 0,
                    min_latency_ms: decodedMetrics.latencies?.min ? decodedMetrics.latencies.min / 1000000 : 0,
                    max_latency_ms: decodedMetrics.latencies?.max ? decodedMetrics.latencies.max / 1000000 : 0,
                    throughput: decodedMetrics.throughput || 0,
                    rate: decodedMetrics.rate || 0,
                    duration_ns: decodedMetrics.duration || 0,
                    bytes_in: decodedMetrics.bytes_in || {},
                    bytes_out: decodedMetrics.bytes_out || {},
                    errors: decodedMetrics.errors || [],
                    status_codes: decodedMetrics.status_codes || {},
                    decoded: true,
                    vegeta_format: true
                };
            }

            // Handle direct metrics format
            return {
                ...result,
                total_requests: decodedMetrics.total_requests || decodedMetrics.requests || result.total_requests || 0,
                completed_requests: decodedMetrics.completed_requests || decodedMetrics.successful_requests || result.completed_requests || 0,
                success_rate: decodedMetrics.success_rate || result.success_rate || 0,
                average_latency_ms: decodedMetrics.average_latency_ms || decodedMetrics.avg_latency || result.average_latency_ms || 0,
                p95_latency_ms: decodedMetrics.p95_latency_ms || decodedMetrics.p95_latency || result.p95_latency_ms || 0,
                errors: decodedMetrics.errors || [],
                status_codes: decodedMetrics.status_codes || {},
                decoded: true
            };
        }

        // Use raw API fields as fallback
        return {
            ...result,
            total_requests: result.totalRequests || result.total_requests || 0,
            completed_requests: result.completedRequests || result.completed_requests || 0,
            success_rate: result.successRate || result.success_rate || 0,
            average_latency_ms: result.averageLatencyMs || result.average_latency_ms || 0,
            p95_latency_ms: result.p95LatencyMs || result.p95_latency_ms || 0,
            p50_latency_ms: 0,
            p90_latency_ms: 0,
            p99_latency_ms: 0,
            min_latency_ms: 0,
            max_latency_ms: 0,
            throughput: 0,
            rate: 0,
            duration_ns: result.durationMs ? result.durationMs * 1000000 : 0,
            status_codes: result.statusCodes || result.status_codes || {},
            errors: [],
            bytes_in: {},
            bytes_out: {},
            decoded: false,
            vegeta_format: false
        };
    } catch (error) {
        // Log error for debugging but don't crash the UI
        return { ...result, decoded: false };
    }
};

export const TestDetailPage = () => {
    const { testId } = useParams();
    const navigate = useNavigate();
    const [testDetail, setTestDetail] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const [activeTab, setActiveTab] = useState('overview');

    useEffect(() => {
        const fetchData = async () => {
            if (!testId) return;

            setLoading(true);
            setError('');
            try {
                const data = await fetchTestDetail(testId);
                setTestDetail(data);
            } catch (err) {
                // Log error for debugging but don't crash the UI
                setError(err.message);
            } finally {
                setLoading(false);
            }
        };

        fetchData();
    }, [testId]);

    const handleReplay = async () => {
        try {
            await replayTest(testId, testDetail?.test?.name);
            alert('Test replayed successfully!');
            navigate('/test-history');
        } catch (err) {
            alert(`Failed to replay test: ${err.message}`);
        }
    };

    // Prepare chart data
    const workerPerformanceData = testDetail?.results?.map((result, index) => {
        const decodedResult = decodeWorkerMetrics(result);
        return {
            name: `Worker ${index + 1}`,
            requests: decodedResult.total_requests || 0,
            completed: decodedResult.completed_requests || 0,
            avgLatency: decodedResult.average_latency_ms || 0,
            p95Latency: decodedResult.p95_latency_ms || 0,
            successRate: ((decodedResult.success_rate || 0) * 100)
        };
    }) || [];

    // Aggregate status codes from all worker results
    const statusCodeData = testDetail?.results ? (() => {
        const aggregatedStatusCodes = {};

        testDetail.results.forEach(result => {
            const decodedResult = decodeWorkerMetrics(result);
            if (decodedResult.status_codes) {
                Object.entries(decodedResult.status_codes).forEach(([code, count]) => {
                    aggregatedStatusCodes[code] = (aggregatedStatusCodes[code] || 0) + count;
                });
            }
        });

        return Object.entries(aggregatedStatusCodes).map(([code, count]) => ({
            name: `HTTP ${code}`,
            value: count || 0
        }));
    })() : [];

    if (loading) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <LoadingSpinner size="large" />
            </div>
        );
    }

    if (error) {
        return (
            <div className="min-h-screen bg-gray-50 p-6">
                <div className="max-w-7xl mx-auto">
                    <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
                        {error}
                    </div>
                </div>
            </div>
        );
    }

    if (!testDetail) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <div className="text-gray-500">No test data available</div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50">
            {/* Header */}
            <div className="bg-white shadow-sm border-b border-gray-200">
                <div className="max-w-7xl mx-auto px-6 py-4">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center space-x-4">
                            <button
                                onClick={() => navigate('/test-history')}
                                className="flex items-center space-x-2 text-gray-600 hover:text-gray-900 transition-colors"
                            >
                                <ArrowLeft className="w-5 h-5" />
                                <span>Back to Tests</span>
                            </button>
                            <div className="h-6 w-px bg-gray-300"></div>
                            <div>
                                <h1 className="text-2xl font-bold text-gray-900">Test Details</h1>
                                {testDetail?.test && (
                                    <p className="text-gray-600 mt-1">{testDetail.test.name}</p>
                                )}
                            </div>
                        </div>
                        <button
                            onClick={handleReplay}
                            className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-all"
                        >
                            <Play className="w-4 h-4" />
                            <span>Replay Test</span>
                        </button>
                    </div>
                </div>
            </div>

            {/* Tabs */}
            <div className="bg-white border-b border-gray-200">
                <div className="max-w-7xl mx-auto px-6">
                    <nav className="flex space-x-8">
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
            </div>

            {/* Content */}
            <div className="max-w-7xl mx-auto px-6 py-8">
                <PageContent
                    activeTab={activeTab}
                    testDetail={testDetail}
                    workerPerformanceData={workerPerformanceData}
                    statusCodeData={statusCodeData}
                />
            </div>
        </div>
    );
};

const PageContent = ({ activeTab, testDetail, workerPerformanceData, statusCodeData }) => {
    const test = testDetail?.test;

    try {
        if (activeTab === 'overview') {
            return <OverviewTab test={test} />;
        }

        if (activeTab === 'results') {
            return <ResultsTab testDetail={testDetail} />;
        }

        if (activeTab === 'charts') {
            return <ChartsTab workerPerformanceData={workerPerformanceData} statusCodeData={statusCodeData} />;
        }

        return <div className="text-gray-500 text-center py-8">Invalid tab selected</div>;
    } catch (error) {
        // Log error for debugging but don't crash the UI
        return (
            <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
                Error rendering content: {error.message}
            </div>
        );
    }
};

// Reuse the same components from the modal with full styling
const OverviewTab = ({ test }) => (
    <div className="space-y-8">
        {/* Test Info Cards */}
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
        <div className="bg-white p-8 rounded-xl shadow-sm border border-gray-200">
            <h3 className="text-xl font-semibold text-gray-900 mb-6">Test Status</h3>
            <div className="flex items-center justify-between mb-6">
                <StatusBadge status={test.status} />
                <div className="text-sm text-gray-600">
                    Created: {formatDate(getValueWithFallback(test, ['createdAt', 'created_at', 'created']))}
                </div>
            </div>

            {test.assigned_workers_ids?.length > 0 && (
                <div>
                    <div className="text-sm font-medium text-gray-700 mb-3">Assigned Workers:</div>
                    <div className="flex flex-wrap gap-2">
                        {test.assigned_workers_ids.map((workerId) => (
                            <span key={workerId} className="px-3 py-1 bg-blue-100 text-blue-800 rounded-full text-sm">
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
    <div className="space-y-8">
        {testDetail?.aggregated_result ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                <InfoCard
                    value={(testDetail.aggregated_result.total_requests || 0).toLocaleString()}
                    label="Total Requests"
                    bgColor="bg-blue-50"
                    textColor="text-blue-900"
                />
                <InfoCard
                    value={(testDetail.aggregated_result.successful_requests || 0).toLocaleString()}
                    label="Successful"
                    bgColor="bg-green-50"
                    textColor="text-green-900"
                />
                <InfoCard
                    value={(testDetail.aggregated_result.failed_requests || 0).toLocaleString()}
                    label="Failed"
                    bgColor="bg-red-50"
                    textColor="text-red-900"
                />
                <InfoCard
                    value={`${(testDetail.aggregated_result.avg_latency_ms || 0).toFixed(2)}ms`}
                    label="Avg Latency"
                    bgColor="bg-yellow-50"
                    textColor="text-yellow-900"
                />
            </div>
        ) : (
            <div className="bg-white p-8 rounded-xl shadow-sm border border-gray-200">
                <div className="text-gray-500 text-center py-8">
                    No aggregated results available yet.
                </div>
            </div>
        )}

        {/* Individual Worker Results */}
        {testDetail?.results?.length > 0 && (
            <div className="bg-white rounded-xl shadow-sm border border-gray-200">
                <WorkerResultsTable results={testDetail.results} />
            </div>
        )}
    </div>
);

const ChartsTab = ({ workerPerformanceData, statusCodeData }) => {
    return (
        <div className="space-y-8">
            {workerPerformanceData && workerPerformanceData.length > 0 ? (
                <>
                    {/* Worker Performance Chart */}
                    <ChartContainer title="Worker Performance">
                        <ResponsiveContainer width="100%" height={400}>
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
                        <ResponsiveContainer width="100%" height={400}>
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
            ) : (
                <div className="bg-white p-8 rounded-xl shadow-sm border border-gray-200">
                    <div className="text-gray-500 text-center py-8">
                        No worker performance data available for charts.
                    </div>
                </div>
            )}

            {/* Status Code Distribution */}
            {statusCodeData && statusCodeData.length > 0 ? (
                <ChartContainer title="Status Code Distribution">
                    <ResponsiveContainer width="100%" height={400}>
                        <PieChart>
                            <Pie
                                data={statusCodeData}
                                cx="50%"
                                cy="50%"
                                labelLine={false}
                                label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                                outerRadius={100}
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
            ) : (
                <div className="bg-white p-8 rounded-xl shadow-sm border border-gray-200">
                    <div className="text-gray-500 text-center py-8">
                        No status code data available for pie chart.
                    </div>
                </div>
            )}

            {(!workerPerformanceData || workerPerformanceData.length === 0) &&
             (!statusCodeData || statusCodeData.length === 0) && (
                <div className="bg-white p-8 rounded-xl shadow-sm border border-gray-200">
                    <div className="text-gray-500 text-center py-12">
                        No chart data available. Results may still be processing.
                    </div>
                </div>
            )}
        </div>
    );
};

const InfoCard = ({ icon: Icon, value, label, bgColor, textColor, iconColor }) => (
    <div className={`${bgColor} p-6 rounded-xl shadow-sm border border-gray-200`}>
        <div className="flex items-center space-x-4">
            {Icon && <Icon className={`w-10 h-10 ${iconColor}`} />}
            <div>
                <div className={`text-3xl font-bold ${textColor}`}>
                    {value}
                </div>
                <div className={`text-sm ${iconColor} mt-1`}>{label}</div>
            </div>
        </div>
    </div>
);

const WorkerResultsTable = ({ results }) => {
    const decodedResults = results.map(result => decodeWorkerMetrics(result));

    return (
        <div className="p-6">
            <h3 className="text-xl font-semibold text-gray-900 mb-6">Individual Worker Results</h3>
            <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Worker</th>
                            <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Total Requests</th>
                            <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Success Rate</th>
                            <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Avg Latency</th>
                            <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">P50 Latency</th>
                            <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">P95 Latency</th>
                            <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Rate (req/s)</th>
                            <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                        {decodedResults.map((result, index) => {
                            const successRate = typeof result.success_rate === 'number'
                                ? result.success_rate
                                : (result.completed_requests / Math.max(result.total_requests, 1));

                            return (
                                <tr key={result.id || index} className="hover:bg-gray-50">
                                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                                        <div className="flex items-center space-x-2">
                                            <span>Worker {index + 1}</span>
                                            {result.decoded && (
                                                <span className="inline-block w-2 h-2 bg-green-500 rounded-full" title="Metrics decoded from base64"></span>
                                            )}
                                            {result.vegeta_format && (
                                                <span className="text-xs bg-purple-100 text-purple-800 px-1 rounded" title="Vegeta format">V</span>
                                            )}
                                        </div>
                                        {result.worker_id && (
                                            <div className="text-xs text-gray-500">{result.worker_id}</div>
                                        )}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        <div>{(result.total_requests || 0).toLocaleString()}</div>
                                        {result.vegeta_format && result.errors?.length > 0 && (
                                            <div className="text-xs text-red-600">{result.errors.length} errors</div>
                                        )}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        <span className={`px-2 py-1 rounded-full text-xs ${
                                            successRate >= 0.95 ? 'bg-green-100 text-green-800' :
                                            successRate >= 0.8 ? 'bg-yellow-100 text-yellow-800' :
                                            'bg-red-100 text-red-800'
                                        }`}>
                                            {(successRate * 100).toFixed(1)}%
                                        </span>
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {(result.average_latency_ms || 0).toFixed(2)}ms
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {(result.p50_latency_ms || 0).toFixed(2)}ms
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {(result.p95_latency_ms || 0).toFixed(2)}ms
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {(result.rate || result.throughput || 0).toFixed(2)}
                                    </td>
                                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        <StatusBadge status={result.status || 'completed'} />
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>

            {/* Additional metrics if available */}
            {decodedResults.some(result => result.errors?.length > 0 || Object.keys(result.status_codes || {}).length > 0 || result.vegeta_format) && (
                <div className="mt-8">
                    <h4 className="text-lg font-medium text-gray-900 mb-4">Additional Details</h4>
                    <div className="space-y-4">
                        {decodedResults.map((result, index) => (
                            <div key={index} className="bg-gray-50 p-6 rounded-lg">
                                <div className="text-sm font-medium text-gray-700 mb-3 flex items-center space-x-2">
                                    <span>Worker {index + 1}</span>
                                    {result.vegeta_format && (
                                        <span className="text-xs bg-purple-100 text-purple-800 px-2 py-1 rounded">Vegeta Metrics</span>
                                    )}
                                </div>

                                {/* Vegeta-specific metrics */}
                                {result.vegeta_format && (
                                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-3">
                                        <div>
                                            <span className="text-xs font-medium text-gray-600">Throughput: </span>
                                            <span className="text-xs text-gray-800">{(result.throughput || 0).toFixed(2)} req/s</span>
                                        </div>
                                        <div>
                                            <span className="text-xs font-medium text-gray-600">Duration: </span>
                                            <span className="text-xs text-gray-800">{((result.duration_ns || 0) / 1000000000).toFixed(2)}s</span>
                                        </div>
                                        <div>
                                            <span className="text-xs font-medium text-gray-600">Min Latency: </span>
                                            <span className="text-xs text-gray-800">{(result.min_latency_ms || 0).toFixed(2)}ms</span>
                                        </div>
                                        <div>
                                            <span className="text-xs font-medium text-gray-600">Max Latency: </span>
                                            <span className="text-xs text-gray-800">{(result.max_latency_ms || 0).toFixed(2)}ms</span>
                                        </div>
                                        {result.p90_latency_ms > 0 && (
                                            <div>
                                                <span className="text-xs font-medium text-gray-600">P90 Latency: </span>
                                                <span className="text-xs text-gray-800">{result.p90_latency_ms.toFixed(2)}ms</span>
                                            </div>
                                        )}
                                        {result.p99_latency_ms > 0 && (
                                            <div>
                                                <span className="text-xs font-medium text-gray-600">P99 Latency: </span>
                                                <span className="text-xs text-gray-800">{result.p99_latency_ms.toFixed(2)}ms</span>
                                            </div>
                                        )}
                                        {result.bytes_in?.total > 0 && (
                                            <div>
                                                <span className="text-xs font-medium text-gray-600">Bytes In: </span>
                                                <span className="text-xs text-gray-800">{(result.bytes_in.total || 0).toLocaleString()} bytes</span>
                                            </div>
                                        )}
                                        {result.bytes_out?.total > 0 && (
                                            <div>
                                                <span className="text-xs font-medium text-gray-600">Bytes Out: </span>
                                                <span className="text-xs text-gray-800">{(result.bytes_out.total || 0).toLocaleString()} bytes</span>
                                            </div>
                                        )}
                                    </div>
                                )}

                                {/* Status Codes */}
                                {Object.keys(result.status_codes || {}).length > 0 && (
                                    <div className="mb-2">
                                        <span className="text-xs font-medium text-gray-600">Status Codes: </span>
                                        {Object.entries(result.status_codes).map(([code, count]) => (
                                            <span key={code} className={`inline-block text-xs px-2 py-1 rounded mr-2 ${
                                                code.startsWith('2') ? 'bg-green-100 text-green-800' :
                                                code.startsWith('4') ? 'bg-yellow-100 text-yellow-800' :
                                                code.startsWith('5') ? 'bg-red-100 text-red-800' :
                                                'bg-blue-100 text-blue-800'
                                            }`}>
                                                {code}: {count}
                                            </span>
                                        ))}
                                    </div>
                                )}

                                {/* Errors */}
                                {result.errors?.length > 0 && (
                                    <div>
                                        <span className="text-xs font-medium text-gray-600">Errors: </span>
                                        <span className="text-xs text-red-600">{result.errors.length} error(s)</span>
                                        {result.errors.length <= 5 && (
                                            <div className="mt-1">
                                                {result.errors.map((error, errorIndex) => (
                                                    <div key={errorIndex} className="text-xs text-red-500 bg-red-50 p-1 rounded mt-1">
                                                        {typeof error === 'string' ? error : JSON.stringify(error)}
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
};

const ChartContainer = ({ title, children }) => (
    <div className="bg-white p-8 rounded-xl shadow-sm border border-gray-200">
        <h3 className="text-xl font-semibold text-gray-900 mb-6">{title}</h3>
        {children}
    </div>
);
