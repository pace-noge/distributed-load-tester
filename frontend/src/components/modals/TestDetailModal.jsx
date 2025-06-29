import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Play, XCircle, Target, Users, Clock, BarChart3,
    ExternalLink
} from 'lucide-react';

import { LoadingSpinner, StatusBadge } from '../common/UIComponents.jsx';
import { formatDate, formatDuration, formatRateDistribution, getValueWithFallback } from '../../utils/formatters.js';
import { fetchTestDetail, replayTest } from '../../utils/api.js';

export const TestDetailModal = ({ testId, isOpen, onClose }) => {
    const navigate = useNavigate();
    const [testDetail, setTestDetail] = useState(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

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

    const handleViewFullDetails = () => {
        navigate(`/test/${testId}`);
        onClose();
    };

    if (!isOpen) return null;    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-2xl shadow-2xl w-full max-w-4xl max-h-[80vh] overflow-hidden">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-gray-200">
                    <div>
                        <h2 className="text-2xl font-bold text-gray-900">Test Overview</h2>
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
                        <div className="space-y-6">
                            {/* Test Configuration Overview */}
                            <OverviewContent test={testDetail.test} />

                            {/* Quick Results Summary */}
                            {testDetail.aggregated_result && (
                                <QuickResultsSummary aggregatedResult={testDetail.aggregated_result} />
                            )}

                            {/* Action Buttons */}
                            <div className="flex justify-center pt-4">
                                <button
                                    onClick={handleViewFullDetails}
                                    className="flex items-center space-x-2 px-6 py-3 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-all"
                                >
                                    <ExternalLink className="w-4 h-4" />
                                    <span>View Full Details & Charts</span>
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

const OverviewContent = ({ test }) => (
    <div className="space-y-6">
        {/* Test Configuration Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
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

        {/* Status and Basic Info */}
        <div className="bg-gray-50 p-4 rounded-lg">
            <h3 className="text-lg font-medium text-gray-900 mb-3">Test Status</h3>
            <div className="flex items-center justify-between mb-3">
                <StatusBadge status={test.status} />
                <div className="text-sm text-gray-600">
                    Created: {formatDate(getValueWithFallback(test, ['createdAt', 'created_at', 'created']))}
                </div>
            </div>

            {test.assigned_workers_ids?.length > 0 && (
                <div>
                    <div className="text-sm font-medium text-gray-700 mb-2">Assigned Workers:</div>
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

const QuickResultsSummary = ({ aggregatedResult }) => (
    <div className="bg-gray-50 p-4 rounded-lg">
        <h3 className="text-lg font-medium text-gray-900 mb-3">Quick Results Summary</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
                <div className="text-2xl font-bold text-blue-600">
                    {(aggregatedResult.total_requests || 0).toLocaleString()}
                </div>
                <div className="text-sm text-gray-600">Total Requests</div>
            </div>
            <div className="text-center">
                <div className="text-2xl font-bold text-green-600">
                    {(aggregatedResult.successful_requests || 0).toLocaleString()}
                </div>
                <div className="text-sm text-gray-600">Successful</div>
            </div>
            <div className="text-center">
                <div className="text-2xl font-bold text-red-600">
                    {(aggregatedResult.failed_requests || 0).toLocaleString()}
                </div>
                <div className="text-sm text-gray-600">Failed</div>
            </div>
            <div className="text-center">
                <div className="text-2xl font-bold text-yellow-600">
                    {(aggregatedResult.avg_latency_ms || 0).toFixed(2)}ms
                </div>
                <div className="text-sm text-gray-600">Avg Latency</div>
            </div>
        </div>

        <div className="mt-3 text-center">
            <div className="text-sm text-gray-600">
                View the dedicated page for detailed worker metrics, charts, and comprehensive analysis.
            </div>
        </div>
    </div>
);

const InfoCard = ({ icon: Icon, value, label, bgColor, textColor, iconColor }) => (
    <div className={`${bgColor} p-4 rounded-lg`}>
        <div className="flex items-center space-x-3">
            {Icon && <Icon className={`w-6 h-6 ${iconColor}`} />}
            <div>
                <div className={`text-xl font-bold ${textColor}`}>
                    {value}
                </div>
                <div className={`text-xs ${iconColor}`}>{label}</div>
            </div>
        </div>
    </div>
);
