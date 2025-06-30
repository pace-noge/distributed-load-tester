import { Link, useLocation } from 'react-router-dom';
import { Inbox, Zap } from 'lucide-react';
import { useEffect, useState } from 'react';
import { fetchInbox } from '../../utils/api';

const NAV_ITEMS = [
    { id: 'inbox', label: 'Inbox', icon: Inbox, path: '/inbox' },
];

export const Navbar = ({ inboxRight }) => {
    const location = useLocation();
    const [hasUnread, setHasUnread] = useState(false);

    // Poll inbox for unread status
    useEffect(() => {
        let ws;
        let pollInterval;
        const checkInbox = async () => {
            try {
                const data = await fetchInbox();
                const userId = localStorage.getItem('user_id');
                setHasUnread(
                    (data.inbox || []).some(item => !(item.readBy || []).includes(userId) && !item.isExpired)
                );
            } catch {
                setHasUnread(false);
            }
        };
        checkInbox();
        if (window.WebSocket) {
            const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            ws = new window.WebSocket(wsProtocol + '//' + window.location.host + '/ws');
            ws.onmessage = (event) => {
                try {
                    const msg = JSON.parse(event.data);
                    if (msg.type === 'inbox') {
                        checkInbox();
                    }
                } catch {
                    checkInbox();
                }
            };
        } else {
            pollInterval = setInterval(checkInbox, 10000);
        }
        return () => {
            if (ws) ws.close();
            if (pollInterval) clearInterval(pollInterval);
        };
    }, []);

    return (
        <nav className="bg-white border-b border-gray-200 px-6 py-4">
            <div className="flex items-center justify-between">
                {/* Left: Logo and Title */}
                <div className="flex items-center space-x-3">
                    <div className="h-8 w-8 bg-blue-600 rounded-lg flex items-center justify-center">
                        <Zap className="h-5 w-5 text-white" />
                    </div>
                    <h1 className="text-xl font-bold text-gray-900">Load Tester</h1>
                </div>
                {/* Right: Inbox only */}
                <div className="flex items-center space-x-1">
                    {NAV_ITEMS.map((item) => {
                        const Icon = item.icon;
                        const isActive = location.pathname === item.path;
                        return (
                            <Link
                                key={item.id}
                                to={item.path}
                                className={`flex items-center space-x-2 px-4 py-2 rounded-lg font-medium transition-all ${
                                    isActive
                                        ? 'bg-blue-50 text-blue-700 border border-blue-200'
                                        : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
                                }`}
                            >
                                <Icon className="w-4 h-4" />
                                <span>{item.label}</span>
                                {item.id === 'inbox' && hasUnread && <span className="ml-2 inline-block w-2 h-2 bg-red-500 rounded-full" title="New" />}
                            </Link>
                        );
                    })}
                </div>
            </div>
        </nav>
    );
};
