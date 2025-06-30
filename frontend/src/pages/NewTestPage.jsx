import { useState } from 'react';
import { Rocket } from 'lucide-react';
import { API_BASE_URL } from '../utils/constants';
import { authenticatedFetch } from '../utils/api';
import { LoadingSpinner } from '../components/common/UIComponents';

const NewTestPage = () => {
    const [formData, setFormData] = useState({
        name: '',
        durationSeconds: '30s',
        ratePerSecond: 10,
        workerCount: 1,
        rateDistribution: 'shared',
        rateWeights: [],
        rampDuration: '',
        rampStartDelay: '',
        rampSteps: '',
        targets: '[{"method":"GET","url":"https://httpbin.org/get"}]',
        // Vegeta options
        timeout: 30,
        redirects: 10,
        keepalive: true,
        http2: false,
        insecure: false,
        connections: 10000,
        vegetaPayloadJson: '{}'
    });

    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState('');
    const [success, setSuccess] = useState('');

    const handleInputChange = (e) => {
        const { name, value, type, checked } = e.target;
        setFormData(prev => ({
            ...prev,
            [name]: type === 'checkbox' ? checked : type === 'number' ? Number(value) : value
        }));
    };

    const handleWeightsChange = (index, value) => {
        const newWeights = [...formData.rateWeights];
        newWeights[index] = parseFloat(value) || 0;
        setFormData(prev => ({ ...prev, rateWeights: newWeights }));
    };

    const updateWorkerCount = (count) => {
        const weights = Array(count).fill(1.0);
        setFormData(prev => ({
            ...prev,
            workerCount: count,
            rateWeights: prev.rateDistribution === 'weighted' ? weights : []
        }));
    };

    const handleRateDistributionChange = (distribution) => {
        const weights = distribution === 'weighted' ? Array(formData.workerCount).fill(1.0) : [];
        setFormData(prev => ({
            ...prev,
            rateDistribution: distribution,
            rateWeights: weights
        }));
    };

    const buildVegetaPayload = () => {
        const baseOptions = {
            timeout: formData.timeout,
            redirects: formData.redirects,
            keepalive: formData.keepalive,
            http2: formData.http2,
            insecure: formData.insecure,
            connections: formData.connections,
        };

        let userOptions = {};
        try {
            if (formData.vegetaPayloadJson.trim()) {
                userOptions = JSON.parse(formData.vegetaPayloadJson);
            }
        } catch (e) {
            // If JSON is invalid, use empty object
        }

        // Merge with user options taking precedence
        return JSON.stringify({ ...baseOptions, ...userOptions });
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setSubmitting(true);
        setError('');
        setSuccess('');

        try {
            // Validate targets JSON
            JSON.parse(formData.targets);

            // Build final payload
            const payload = {
                name: formData.name,
                duration_seconds: formData.durationSeconds,
                rate_per_second: formData.ratePerSecond,
                worker_count: formData.workerCount,
                rate_distribution: formData.rateDistribution,
                targets_base64: btoa(formData.targets),
                vegeta_payload_json: buildVegetaPayload(),
            };

            // Add conditional fields
            if (formData.rateDistribution === 'weighted' && formData.rateWeights.length > 0) {
                payload.rate_weights = formData.rateWeights;
            }

            if (formData.rateDistribution === 'ramped') {
                if (formData.rampDuration) payload.ramp_duration = formData.rampDuration;
                if (formData.rampStartDelay) payload.ramp_start_delay = formData.rampStartDelay;
                if (formData.rampSteps) payload.ramp_steps = parseInt(formData.rampSteps);
            }

            await authenticatedFetch(`${API_BASE_URL}/test/submit`, {
                method: 'POST',
                body: JSON.stringify(payload)
            });

            setSuccess('Test submitted successfully!');
            // Reset form
            setFormData({
                name: '',
                durationSeconds: '30s',
                ratePerSecond: 10,
                workerCount: 1,
                rateDistribution: 'shared',
                rateWeights: [],
                rampDuration: '',
                rampStartDelay: '',
                rampSteps: '',
                targets: '[{"method":"GET","url":"https://httpbin.org/get"}]',
                timeout: 30,
                redirects: 10,
                keepalive: true,
                http2: false,
                insecure: false,
                connections: 10000,
                vegetaPayloadJson: '{}'
            });
        } catch (err) {
            setError(err.message);
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <div className="max-w-4xl mx-auto">
            <div className="mb-8">
                <h1 className="text-3xl font-bold text-gray-900">Create New Load Test</h1>
                <p className="text-gray-600 mt-2">Configure and launch a new distributed load test</p>
            </div>

            <form onSubmit={handleSubmit} className="space-y-8">
                {/* Basic Configuration */}
                <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900 mb-4">Basic Configuration</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Test Name</label>
                            <input
                                type="text"
                                name="name"
                                value={formData.name}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                required
                                placeholder="My Load Test"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Duration</label>
                            <input
                                type="text"
                                name="durationSeconds"
                                value={formData.durationSeconds}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                required
                                placeholder="30s, 5m, 1h"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Rate (req/sec)</label>
                            <input
                                type="number"
                                name="ratePerSecond"
                                value={formData.ratePerSecond}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                required
                                min="1"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Worker Count</label>
                            <input
                                type="number"
                                name="workerCount"
                                value={formData.workerCount}
                                onChange={(e) => updateWorkerCount(parseInt(e.target.value))}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                required
                                min="1"
                            />
                        </div>
                    </div>
                </div>

                {/* Rate Distribution */}
                <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900 mb-4">Rate Distribution</h2>
                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Distribution Mode</label>
                            <select
                                name="rateDistribution"
                                value={formData.rateDistribution}
                                onChange={(e) => handleRateDistributionChange(e.target.value)}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            >
                                <option value="shared">Shared - Split rate across workers</option>
                                <option value="same">Same - Each worker uses full rate</option>
                                <option value="weighted">Weighted - Custom weights per worker</option>
                                <option value="ramped">Ramped - Gradually increase rate</option>
                                <option value="burst">Burst - High load on fewer workers</option>
                            </select>
                        </div>

                        {formData.rateDistribution === 'weighted' && (
                            <div>
                                <label className="block text-sm font-medium text-gray-700 mb-2">Worker Weights</label>
                                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                                    {Array(formData.workerCount).fill(0).map((_, index) => (
                                        <div key={index}>
                                            <label className="block text-xs text-gray-600 mb-1">Worker {index + 1}</label>
                                            <input
                                                type="number"
                                                step="0.1"
                                                value={formData.rateWeights[index] || 1.0}
                                                onChange={(e) => handleWeightsChange(index, e.target.value)}
                                                className="w-full px-3 py-2 border border-gray-300 rounded focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                                min="0.1"
                                            />
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {formData.rateDistribution === 'ramped' && (
                            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-2">Ramp Duration</label>
                                    <input
                                        type="text"
                                        name="rampDuration"
                                        value={formData.rampDuration}
                                        onChange={handleInputChange}
                                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                        placeholder="30s (optional)"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-2">Start Delay</label>
                                    <input
                                        type="text"
                                        name="rampStartDelay"
                                        value={formData.rampStartDelay}
                                        onChange={handleInputChange}
                                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                        placeholder="5s (optional)"
                                    />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 mb-2">Steps</label>
                                    <input
                                        type="number"
                                        name="rampSteps"
                                        value={formData.rampSteps}
                                        onChange={handleInputChange}
                                        className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                        placeholder="5 (optional)"
                                        min="1"
                                    />
                                </div>
                            </div>
                        )}
                    </div>
                </div>

                {/* HTTP Configuration */}
                <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900 mb-4">HTTP Configuration</h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Timeout (seconds)</label>
                            <input
                                type="number"
                                name="timeout"
                                value={formData.timeout}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                min="1"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Max Redirects</label>
                            <input
                                type="number"
                                name="redirects"
                                value={formData.redirects}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                min="0"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">Max Connections</label>
                            <input
                                type="number"
                                name="connections"
                                value={formData.connections}
                                onChange={handleInputChange}
                                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                                min="1"
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
                        <label className="flex items-center space-x-3">
                            <input
                                type="checkbox"
                                name="keepalive"
                                checked={formData.keepalive}
                                onChange={handleInputChange}
                                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                            />
                            <span className="text-sm font-medium text-gray-700">Keep-Alive</span>
                        </label>

                        <label className="flex items-center space-x-3">
                            <input
                                type="checkbox"
                                name="http2"
                                checked={formData.http2}
                                onChange={handleInputChange}
                                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                            />
                            <span className="text-sm font-medium text-gray-700">HTTP/2</span>
                        </label>

                        <label className="flex items-center space-x-3">
                            <input
                                type="checkbox"
                                name="insecure"
                                checked={formData.insecure}
                                onChange={handleInputChange}
                                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                            />
                            <span className="text-sm font-medium text-gray-700">Skip TLS Verify</span>
                        </label>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Advanced Vegeta Options (JSON)</label>
                        <textarea
                            name="vegetaPayloadJson"
                            value={formData.vegetaPayloadJson}
                            onChange={handleInputChange}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            rows="3"
                            placeholder='{"headers": {"User-Agent": "MyApp/1.0"}}'
                        />
                        <p className="text-xs text-gray-600 mt-1">Additional Vegeta options in JSON format. Will merge with options above.</p>
                    </div>
                </div>

                {/* Targets */}
                <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900 mb-4">Test Targets</h2>
                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">Targets (JSON)</label>
                        <textarea
                            name="targets"
                            value={formData.targets}
                            onChange={handleInputChange}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                            rows="6"
                            required
                            placeholder='[{"method":"GET","url":"https://example.com/api/endpoint"}]'
                        />
                        <p className="text-xs text-gray-600 mt-1">JSON array of HTTP targets to test</p>
                    </div>
                </div>

                {/* Submit */}
                <div className="flex items-center justify-between">
                    {error && (
                        <div className="bg-red-50 border border-red-200 text-red-800 px-4 py-3 rounded-lg flex-1 mr-4">
                            {error}
                        </div>
                    )}

                    {success && (
                        <div className="bg-green-50 border border-green-200 text-green-800 px-4 py-3 rounded-lg flex-1 mr-4">
                            {success}
                        </div>
                    )}

                    <button
                        type="submit"
                        disabled={submitting}
                        className="flex items-center space-x-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-medium"
                    >
                        {submitting ? <LoadingSpinner size="small" /> : <Rocket className="w-5 h-5" />}
                        <span>{submitting ? 'Submitting...' : 'Launch Test'}</span>
                    </button>
                </div>
            </form>
        </div>
    );
};

export default NewTestPage;
