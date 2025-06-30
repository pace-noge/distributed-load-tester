import { API_BASE_URL } from './constants.js';

/**
 * Authenticated fetch wrapper
 * @param {string} url - The URL to fetch
 * @param {object} options - Fetch options
 * @returns {Promise} - The response data
 */
export const authenticatedFetch = async (url, options = {}) => {
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

/**
 * Fetch test history data with pagination
 * @param {number} page - Page number (1-based)
 * @param {number} limit - Number of items per page
 * @returns {Promise} - Paginated test history data
 */
export const fetchTestHistory = async (page = 1, limit = 20) => {
    const url = `${API_BASE_URL}/tests?page=${page}&limit=${limit}`;
    const data = await authenticatedFetch(url);
    return data;
};

/**
 * Fetch all test history data (backwards compatibility)
 * @returns {Promise} - All test history data
 */
export const fetchAllTestHistory = async () => {
    // For backwards compatibility, fetch all tests by using a large page size
    const data = await fetchTestHistory(1, 1000);
    return Array.isArray(data) ? data : (data.tests || data.data || []);
};

/**
 * Fetch test detail data
 * @param {string} testId - Test ID
 * @returns {Promise} - Combined test detail data
 */
export const fetchTestDetail = async (testId) => {
    // First, get all tests to find the specific test details
    const testsData = await fetchAllTestHistory();
    const tests = Array.isArray(testsData) ? testsData : (testsData.tests || testsData.data || []);
    const testInfo = tests.find(test => test.id === testId);

    if (!testInfo) {
        throw new Error(`Test with ID ${testId} not found`);
    }

    // Fetch test results and aggregated results in parallel
    const [resultsData, aggregatedData] = await Promise.all([
        authenticatedFetch(`${API_BASE_URL}/tests/${testId}/results`)
            .catch(_err => {
                // Failed to fetch test results - this is expected for some tests
                return null;
            }),
        authenticatedFetch(`${API_BASE_URL}/tests/${testId}/aggregated-result`)
            .catch(_err => {
                // Failed to fetch aggregated results - this is expected for some tests
                return null;
            })
    ]);

    // Combine all the data
    return {
        test: testInfo,
        results: resultsData || [],
        aggregated_result: aggregatedData || null
    };
};

/**
 * Replay a test
 * @param {string} testId - Test ID
 * @param {string} testName - Test name
 * @returns {Promise} - Replay response
 */
export const replayTest = async (testId, testName) => {
    // First, get the test details to extract the original configuration
    const testDetail = await fetchTestDetail(testId);
    const originalTest = testDetail.test;

    if (!originalTest) {
        throw new Error('Original test configuration not found');
    }

    // Create a new test request with the same configuration but new name
    // Use snake_case field names to match the backend expectations
    const replayRequest = {
        name: `${testName} (Replay)`,
        vegeta_payload_json: originalTest.vegetaPayloadJson || originalTest.vegetaPayloadJSON,
        duration_seconds: originalTest.durationSeconds?.toString() || originalTest.duration_seconds?.toString(),
        rate_per_second: originalTest.ratePerSecond || originalTest.rate_per_second,
        targets_base64: originalTest.targetsBase64 || originalTest.targets_base64,
        worker_count: originalTest.workerCount || originalTest.worker_count,
        rate_distribution: originalTest.rateDistribution || originalTest.rate_distribution || "same"
    };

    // Submit the new test using the existing submit endpoint
    return await authenticatedFetch(`${API_BASE_URL}/test/submit`, {
        method: 'POST',
        body: JSON.stringify(replayRequest)
    });
};

/**
 * Login user
 * @param {string} username - Username
 * @param {string} password - Password
 * @returns {Promise} - Login response with token
 */
export const loginUser = async (username, password) => {
    const response = await fetch('http://localhost:8080/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Login failed' }));
        throw new Error(errorData.message || 'Login failed');
    }

    return response.json();
};
