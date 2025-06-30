import React, { useState, useEffect } from 'react';
import { Calendar, TrendingUp, Target, Activity, AlertTriangle } from 'lucide-react';
import { getAnalyticsOverview, getTargetAnalytics } from '../utils/api.js';

const AnalyticsPage = () => {
    const [analyticsData, setAnalyticsData] = useState(null);
    const [targetAnalytics, setTargetAnalytics] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [timeRange, setTimeRange] = useState({
        startDate: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0], // 30 days ago
        endDate: new Date().toISOString().split('T')[0] // today
    });

    useEffect(() => {
        fetchAnalyticsData();
    }, [timeRange]);

    const fetchAnalyticsData = async () => {
        try {
            setLoading(true);
            setError(null);

            console.log('Fetching analytics data...', timeRange);

            const [overview, targets] = await Promise.all([
                getAnalyticsOverview(timeRange),
                getTargetAnalytics(timeRange)
            ]);

            console.log('Analytics data fetched:', { overview, targets });

            setAnalyticsData(overview);
            setTargetAnalytics(targets);
        } catch (err) {
            console.error('Failed to fetch analytics:', err);
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const handleTimeRangeChange = (field, value) => {
        setTimeRange(prev => ({
            ...prev,
            [field]: value
        }));
    };

    if (loading) {
        return (
            <div className="p-6">
                <div className="flex items-center justify-center h-64">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
                    <span className="ml-4 text-gray-600">Loading analytics...</span>
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-6">
                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                    <div className="flex items-center">
                        <AlertTriangle className="h-5 w-5 text-red-500 mr-2" />
                        <span className="text-red-700">Error loading analytics: {error}</span>
                    </div>
                    <button
                        onClick={fetchAnalyticsData}
                        className="mt-2 px-4 py-2 bg-red-100 text-red-700 rounded hover:bg-red-200"
                    >
                        Retry
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="p-6 space-y-6">
            {/* Header */}
            <div className="flex justify-between items-center">
                <div>
                    <h1 className="text-3xl font-bold text-gray-900">Analytics & Performance Insights</h1>
                    <p className="text-gray-600 mt-1">Analyze your load testing performance and trends</p>
                </div>

                {/* Time Range Selector */}
                <div className="flex items-center space-x-4">
                    <div className="flex items-center space-x-2">
                        <Calendar className="h-5 w-5 text-gray-500" />
                        <label className="text-sm font-medium text-gray-700">From:</label>
                        <input
                            type="date"
                            value={timeRange.startDate}
                            onChange={(e) => handleTimeRangeChange('startDate', e.target.value)}
                            className="border border-gray-300 rounded px-3 py-2 text-sm"
                        />
                    </div>
                    <div className="flex items-center space-x-2">
                        <label className="text-sm font-medium text-gray-700">To:</label>
                        <input
                            type="date"
                            value={timeRange.endDate}
                            onChange={(e) => handleTimeRangeChange('endDate', e.target.value)}
                            className="border border-gray-300 rounded px-3 py-2 text-sm"
                        />
                    </div>
                </div>
            </div>

            {/* User context note */}
            <div className="bg-blue-50 border border-blue-100 rounded p-3 text-blue-700 text-sm mb-4">
                Analytics reflect <b>only your own tests</b>. If you share a test, it will be visible to others (feature coming soon).
            </div>

            {/* Simple Overview Cards */}
            {analyticsData && (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                    <div className="bg-white rounded-lg shadow p-6">
                        <div className="flex items-center">
                            <div className="p-3 rounded-lg bg-blue-50 text-blue-600">
                                <Activity className="h-6 w-6" />
                            </div>
                            <div className="ml-4">
                                <p className="text-sm font-medium text-gray-600">Total Tests</p>
                                <p className="text-2xl font-bold text-gray-900">{analyticsData.totalTests || 0}</p>
                            </div>
                        </div>
                    </div>

                    <div className="bg-white rounded-lg shadow p-6">
                        <div className="flex items-center">
                            <div className="p-3 rounded-lg bg-green-50 text-green-600">
                                <TrendingUp className="h-6 w-6" />
                            </div>
                            <div className="ml-4">
                                <p className="text-sm font-medium text-gray-600">Total Requests</p>
                                <p className="text-2xl font-bold text-gray-900">{analyticsData.totalRequests?.toLocaleString() || 0}</p>
                            </div>
                        </div>
                    </div>

                    <div className="bg-white rounded-lg shadow p-6">
                        <div className="flex items-center">
                            <div className="p-3 rounded-lg bg-emerald-50 text-emerald-600">
                                <Target className="h-6 w-6" />
                            </div>
                            <div className="ml-4">
                                <p className="text-sm font-medium text-gray-600">Success Rate</p>
                                <p className="text-2xl font-bold text-gray-900">{analyticsData.successRate?.toFixed(1) || 0}%</p>
                            </div>
                        </div>
                    </div>

                    <div className="bg-white rounded-lg shadow p-6">
                        <div className="flex items-center">
                            <div className="p-3 rounded-lg bg-purple-50 text-purple-600">
                                <Activity className="h-6 w-6" />
                            </div>
                            <div className="ml-4">
                                <p className="text-sm font-medium text-gray-600">Avg Response Time</p>
                                <p className="text-2xl font-bold text-gray-900">{analyticsData.averageResponseTime?.toFixed(0) || 0}ms</p>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* Target Analytics Table */}
            {targetAnalytics && targetAnalytics.length > 0 && (
                <div className="bg-white rounded-lg shadow">
                    <div className="px-6 py-4 border-b border-gray-200">
                        <h2 className="text-lg font-semibold text-gray-900">Target Performance Analysis</h2>
                        <p className="text-sm text-gray-600">Performance metrics grouped by target URLs</p>
                    </div>
                    <div className="p-6">
                        <div className="overflow-x-auto">
                            <table className="min-w-full divide-y divide-gray-200">
                                <thead className="bg-gray-50">
                                    <tr>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Target
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Tests
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Total Requests
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Success Rate
                                        </th>
                                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                            Avg Response Time
                                        </th>
                                    </tr>
                                </thead>
                                <tbody className="bg-white divide-y divide-gray-200">
                                    {targetAnalytics.map((target, index) => (
                                        <tr key={index} className="hover:bg-gray-50">
                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                                                {target.target}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                {target.testCount?.toLocaleString()}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                {target.totalRequests?.toLocaleString()}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                                                    target.successRate >= 95 ? 'bg-green-100 text-green-800' :
                                                    target.successRate >= 90 ? 'bg-yellow-100 text-yellow-800' :
                                                    'bg-red-100 text-red-800'
                                                }`}>
                                                    {target.successRate?.toFixed(1)}%
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                                {target.averageResponseTime?.toFixed(0)}ms
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
            )}

            {/* Empty State */}
            {analyticsData && analyticsData.totalTests === 0 && (
                <div className="bg-white rounded-lg shadow p-12 text-center">
                    <Activity className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                    <h3 className="text-lg font-medium text-gray-900 mb-2">No test data available</h3>
                    <p className="text-gray-600">
                        You have not run any load tests yet. Only your own tests are shown here.
                    </p>
                </div>
            )}
        </div>
    );
};

export default AnalyticsPage;
