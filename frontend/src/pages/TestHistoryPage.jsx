import { useState, useEffect, useCallback } from 'react';
import { RefreshCw, History, Eye } from 'lucide-react';

import { LoadingSpinner, StatusBadge } from '../components/common/UIComponents.jsx';
import { Pagination } from '../components/common/Pagination.jsx';
import { TestDetailModal } from '../components/modals/TestDetailModal.jsx';
import { formatDate, formatDuration, formatRateDistribution, getValueWithFallback } from '../utils/formatters.js';
import { fetchTestHistory } from '../utils/api.js';

export const TestHistoryPage = () => {
    const [displayedTests, setDisplayedTests] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');
    const [selectedTest, setSelectedTest] = useState(null);
    const [currentPage, setCurrentPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [totalTests, setTotalTests] = useState(0);
    const [itemsPerPage] = useState(10); // Items per page
    const [pageChanging, setPageChanging] = useState(false);    const handleFetchTestHistory = useCallback(async (page = currentPage) => {
        setLoading(true);
        setError('');
        try {
            const response = await fetchTestHistory(page, itemsPerPage);
            
            // Handle the new paginated response format
            if (response.tests && response.pagination) {
                setDisplayedTests(response.tests);
                setTotalTests(response.pagination.total);
                setTotalPages(response.pagination.total_pages);
                setCurrentPage(response.pagination.page);
            } else {
                // Fallback for backwards compatibility
                const tests = Array.isArray(response) ? response : (response.tests || response.data || []);
                setDisplayedTests(tests);
                setTotalTests(tests.length);
                setTotalPages(1);
            }
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }, [itemsPerPage, currentPage]);

    // Remove the useEffect that was handling client-side pagination
    // since we now get paginated data from the server

    useEffect(() => {
        handleFetchTestHistory();
    }, [handleFetchTestHistory]);

    const handlePageChange = (page) => {
        if (page === currentPage) return; // Avoid unnecessary calls
        
        setPageChanging(true);
        setCurrentPage(page);
        
        // Fetch new page data
        handleFetchTestHistory(page);
        
        // Scroll to top of table when changing pages
        window.scrollTo({ top: 0, behavior: 'smooth' });
        
        // Reset page changing state after a brief delay
        setTimeout(() => setPageChanging(false), 300);
    };

    return (
        <div className="space-y-6">
            <TestHistoryHeader
                onRefresh={() => handleFetchTestHistory(currentPage)}
                totalTests={totalTests}
                currentPage={currentPage}
                totalPages={totalPages}
                itemsPerPage={itemsPerPage}
            />

            {error && <ErrorMessage message={error} />}

            {loading ? (
                <LoadingState />
            ) : displayedTests.length === 0 ? (
                <EmptyState />
            ) : (
                <>
                    <TestHistoryTable
                        tests={displayedTests}
                        onViewTest={setSelectedTest}
                        currentPage={currentPage}
                        itemsPerPage={itemsPerPage}
                        isLoading={pageChanging}
                    />
                    {totalPages > 1 && (
                        <div className="space-y-4">
                            <PaginationSummary
                                currentPage={currentPage}
                                totalItems={totalTests}
                                itemsPerPage={itemsPerPage}
                            />
                            <Pagination
                                currentPage={currentPage}
                                totalPages={totalPages}
                                onPageChange={handlePageChange}
                                className="mt-6"
                            />
                        </div>
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

const TestHistoryHeader = ({ onRefresh, totalTests, currentPage, totalPages, itemsPerPage }) => {
    const startItem = totalTests > 0 ? (currentPage - 1) * itemsPerPage + 1 : 0;
    const endItem = Math.min(currentPage * itemsPerPage, totalTests);

    return (
        <div className="flex items-center justify-between">
            <div>
                <h1 className="text-3xl font-bold text-gray-900">Test History</h1>
                <p className="text-gray-600 mt-2">
                    {totalTests > 0 ? (
                        <>
                            Showing {startItem}-{endItem} of {totalTests} tests
                            {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
                        </>
                    ) : (
                        'View and manage your load test history'
                    )}
                </p>
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
};

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
        <p className="text-sm text-gray-500 mt-2">Create your first test to get started!</p>
    </div>
);

const TestHistoryTable = ({ tests, onViewTest, currentPage, itemsPerPage, isLoading }) => (
    <div className="bg-white shadow-sm rounded-lg overflow-hidden relative">
        <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                    <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">#</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Test Name</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Configuration</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                    </tr>
                </thead>
                <tbody className={`bg-white divide-y divide-gray-200 ${isLoading ? 'opacity-50 transition-opacity' : ''}`}>
                    {tests.map((test, index) => (
                        <TestHistoryRow
                            key={test.id}
                            test={test}
                            index={(currentPage - 1) * itemsPerPage + index + 1}
                            onViewTest={onViewTest}
                        />
                    ))}
                </tbody>
            </table>
        </div>
        {isLoading && (
            <div className="absolute inset-0 bg-white bg-opacity-75 flex items-center justify-center">
                <LoadingSpinner size="default" />
            </div>
        )}
    </div>
);

const TestHistoryRow = ({ test, index, onViewTest }) => (
    <tr className="hover:bg-gray-50">
        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            {index}
        </td>
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
        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
            <TestActions
                testId={test.id}
                onViewTest={onViewTest}
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

const TestActions = ({ testId, onViewTest }) => (
    <button
        onClick={() => onViewTest(testId)}
        className="text-blue-600 hover:text-blue-900 flex items-center space-x-1 transition-colors"
    >
        <Eye className="w-4 h-4" />
        <span>View Details</span>
    </button>
);

// Add a component to show pagination summary
const PaginationSummary = ({ currentPage, totalItems, itemsPerPage }) => {
    const startItem = totalItems > 0 ? (currentPage - 1) * itemsPerPage + 1 : 0;
    const endItem = Math.min(currentPage * itemsPerPage, totalItems);

    return (
        <div className="text-sm text-gray-600 bg-gray-50 px-4 py-2 rounded-lg">
            Showing {startItem}-{endItem} of {totalItems} tests
        </div>
    );
};
