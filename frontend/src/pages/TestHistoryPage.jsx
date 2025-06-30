import { useState, useEffect, useCallback } from 'react';
import { RefreshCw, History, Eye, Play } from 'lucide-react';

import { LoadingSpinner, StatusBadge } from '../components/common/UIComponents.jsx';
import { Pagination } from '../components/common/Pagination.jsx';
import { TestDetailModal } from '../components/modals/TestDetailModal.jsx';
import { formatDate, formatDuration, formatRateDistribution, getValueWithFallback } from '../utils/formatters.js';
import { fetchTestHistory, replayTest } from '../utils/api.js';

export const TestHistoryPage = () => {
    const [tests, setTests] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [selectedTest, setSelectedTest] = useState(null);
    const [currentPage, setCurrentPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);

    const handleFetchTestHistory = useCallback(async (page = 1) => {
        setLoading(true);
        setError('');
        try {
            const data = await fetchTestHistory(page);
            setTests(data);
            setTotalPages(1); // Adjust based on your API response structure
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        handleFetchTestHistory(currentPage);
    }, [currentPage, handleFetchTestHistory]);

    const handlePageChange = (page) => {
        setCurrentPage(page);
    };

    const handleReplay = async (testId, testName) => {
        try {
            await replayTest(testId, testName);
            alert('Test replayed successfully!');
            handleFetchTestHistory(currentPage);
        } catch (err) {
            alert(`Failed to replay test: ${err.message}`);
        }
    };

    return (
        <div className="space-y-6">
            <TestHistoryHeader onRefresh={() => handleFetchTestHistory(currentPage)} />

            {error && <ErrorMessage message={error} />}

            {loading ? (
                <LoadingState />
            ) : tests.length === 0 ? (
                <EmptyState />
            ) : (
                <>
                    <TestHistoryTable
                        tests={tests}
                        onViewTest={setSelectedTest}
                        onReplayTest={handleReplay}
                    />
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

const TestHistoryHeader = ({ onRefresh }) => (
    <div className="flex items-center justify-between">
        <div>
            <h1 className="text-3xl font-bold text-gray-900">Test History</h1>
            <p className="text-gray-600 mt-2">View and manage your load test history</p>
        </div>
        <button
            onClick={onRefresh}
            className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-all"
        >
            <RefreshCw className="w-4 h-4" />
            <span>Refresh</span>
        </button>
    </div>
);

const ErrorMessage = ({ message }) => (
    <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg">
        {message}
    </div>
);

const LoadingState = () => (
    <div className="flex items-center justify-center py-12">
        <LoadingSpinner size="large" />
    </div>
);

const EmptyState = () => (
    <div className="text-center py-12">
        <History className="w-12 h-12 text-gray-400 mx-auto mb-4" />
        <h3 className="text-lg font-medium text-gray-900 mb-2">No tests found</h3>
        <p className="text-gray-600">You haven&apos;t run any load tests yet.</p>
    </div>
);

const TestHistoryTable = ({ tests, onViewTest, onReplayTest }) => (
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
                        <TestHistoryRow
                            key={test.id}
                            test={test}
                            onViewTest={onViewTest}
                            onReplayTest={onReplayTest}
                        />
                    ))}
                </tbody>
            </table>
        </div>
    </div>
);

const TestHistoryRow = ({ test, onViewTest, onReplayTest }) => (
    <tr className="hover:bg-gray-50">
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
            <TestConfiguration test={test} />
        </td>
        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            {formatDate(getValueWithFallback(test, ['createdAt', 'created_at', 'created']))}
        </td>
        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium space-x-2">
            <TestActions
                testId={test.id}
                testName={test.name}
                onViewTest={onViewTest}
                onReplayTest={onReplayTest}
            />
        </td>
    </tr>
);

const TestConfiguration = ({ test }) => (
    <div className="space-y-1">
        <div>
            {getValueWithFallback(test, ['ratePerSecond', 'rate_per_second'])} req/s × {getValueWithFallback(test, ['workerCount', 'worker_count'], 1)} workers
        </div>
        <div className="text-xs text-gray-400">
            {formatDuration(getValueWithFallback(test, ['durationSeconds', 'duration_seconds', 'duration']))} • {formatRateDistribution(getValueWithFallback(test, ['rateDistribution', 'rate_distribution']))}
        </div>
    </div>
);

const TestActions = ({ testId, testName, onViewTest, onReplayTest }) => (
    <>
        <button
            onClick={() => onViewTest(testId)}
            className="text-blue-600 hover:text-blue-900 flex items-center space-x-1"
        >
            <Eye className="w-4 h-4" />
            <span>View</span>
        </button>
        <button
            onClick={() => onReplayTest(testId, testName)}
            className="text-green-600 hover:text-green-900 flex items-center space-x-1"
        >
            <Play className="w-4 h-4" />
            <span>Replay</span>
        </button>
    </>
);
