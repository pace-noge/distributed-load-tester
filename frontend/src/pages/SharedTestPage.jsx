import { useEffect, useState } from 'react';
import { fetchSharedTest } from '../utils/api';
import { useParams } from 'react-router-dom';

export const SharedTestPage = () => {
  const { linkId } = useParams();
  const [test, setTest] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    setLoading(true);
    fetchSharedTest(linkId)
      .then(setTest)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [linkId]);

  if (loading) return <div>Loading shared test...</div>;
  if (error) return <div className="text-red-500">{error}</div>;
  if (!test) return <div>No test found.</div>;

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h2 className="text-2xl font-bold mb-4">Shared Test</h2>
      <pre className="bg-gray-100 p-4 rounded overflow-x-auto">{JSON.stringify(test, null, 2)}</pre>
    </div>
  );
};
