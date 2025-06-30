import { useEffect, useState } from 'react';
import { fetchInbox, markInboxItemRead } from '../utils/api';
import { useNavigate } from 'react-router-dom';

export const InboxPage = () => {
  const [inbox, setInbox] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    setLoading(true);
    fetchInbox()
      .then((data) => setInbox(data.inbox || []))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const handleOpenTest = async (linkId, testId, isRead) => {
    if (!isRead) {
      await markInboxItemRead(linkId);
    }
    navigate(`/shared/${linkId}`);
  };

  if (loading) return <div>Loading inbox...</div>;
  if (error) return <div className="text-red-500">{error}</div>;

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h2 className="text-2xl font-bold mb-4">Inbox</h2>
      {inbox.length === 0 ? (
        <div className="text-gray-500">No shared tests yet.</div>
      ) : (
        <ul className="space-y-4">
          {inbox.map((item) => {
            const isExpired = item.isExpired;
            const isRead = (item.readBy || []).includes(item.userId);
            return (
              <li key={item.id} className={`p-4 rounded shadow flex items-center justify-between ${isExpired ? 'bg-gray-100' : 'bg-white'}`}>
                <div>
                  <div className="font-semibold">Shared Test: {item.testId}</div>
                  <div className="text-xs text-gray-500">Shared by: {item.sharedBy}</div>
                  <div className="text-xs text-gray-500">Expires: {new Date(item.expiresAt).toLocaleString()}</div>
                  {isExpired && <span className="text-red-500 text-xs">Expired</span>}
                </div>
                <button
                  className={`ml-4 px-4 py-2 rounded ${isRead ? 'bg-gray-300' : 'bg-blue-500 text-white'} ${isExpired ? 'cursor-not-allowed' : ''}`}
                  disabled={isExpired}
                  onClick={() => handleOpenTest(item.id, item.testId, isRead)}
                >
                  {isRead ? 'View Again' : 'View'}
                  {!isRead && <span className="ml-2 inline-block w-2 h-2 bg-red-500 rounded-full" />}
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
};
