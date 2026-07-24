import type { Chat, Contact } from './types.js';

export const ME_CONTACT: Contact = {
    id: 'user_me',
    name: 'You',
    avatarInitials: 'ME',
    isGroup: false,
};

export const ALICE_CONTACT: Contact = {
    id: 'contact_alice',
    name: 'Alice Smith',
    phoneNumber: '+1 555-0192',
    avatarInitials: 'AS',
    isGroup: false,
};

export const BOB_CONTACT: Contact = {
    id: 'contact_bob',
    name: 'Bob Jones',
    phoneNumber: '+1 555-0144',
    avatarInitials: 'BJ',
    isGroup: false,
};

export const GROUP_DEV_CONTACT: Contact = {
    id: 'contact_group_dev',
    name: 'whatsd Core Team',
    avatarInitials: 'WT',
    isGroup: true,
};

export const CAROL_CONTACT: Contact = {
    id: 'contact_carol',
    name: 'Carol White',
    phoneNumber: '+1 555-0188',
    avatarInitials: 'CW',
    isGroup: false,
};

export const MOCK_CHATS: Chat[] = [
    {
        id: 'chat_alice',
        name: 'Alice Smith',
        avatarInitials: 'AS',
        isGroup: false,
        unreadCount: 2,
        updatedAt: '10:42 AM',
        onlineStatus: 'Online',
        messages: [
            {
                id: 'm1',
                chatId: 'chat_alice',
                sender: ALICE_CONTACT,
                content: 'Hey! Is the whatsd daemon ready for testing on Linux?',
                timestamp: '10:38 AM',
                isOutgoing: false,
                status: 'read',
            },
            {
                id: 'm2',
                chatId: 'chat_alice',
                sender: ME_CONTACT,
                content: 'Yes! The Unix domain socket IPC and LibAdwaita UI are being refined.',
                timestamp: '10:40 AM',
                isOutgoing: true,
                status: 'read',
            },
            {
                id: 'm3',
                chatId: 'chat_alice',
                sender: ALICE_CONTACT,
                content: 'Awesome! Native GNOME components feel so smooth.',
                timestamp: '10:42 AM',
                isOutgoing: false,
                status: 'read',
            },
        ],
    },
    {
        id: 'chat_dev_group',
        name: 'whatsd Core Team',
        avatarInitials: 'WT',
        isGroup: true,
        unreadCount: 5,
        updatedAt: '09:15 AM',
        onlineStatus: '4 members',
        messages: [
            {
                id: 'm4',
                chatId: 'chat_dev_group',
                sender: BOB_CONTACT,
                content: 'whatsmeow backend integration is working great with Go 1.22.',
                timestamp: '09:00 AM',
                isOutgoing: false,
                status: 'read',
            },
            {
                id: 'm5',
                chatId: 'chat_dev_group',
                sender: CAROL_CONTACT,
                content: 'Blueprint templates compile directly with blueprint-compiler.',
                timestamp: '09:12 AM',
                isOutgoing: false,
                status: 'read',
            },
            {
                id: 'm6',
                chatId: 'chat_dev_group',
                sender: ME_CONTACT,
                content: 'Next step: connecting gjsify client over Unix socket.',
                timestamp: '09:15 AM',
                isOutgoing: true,
                status: 'delivered',
            },
        ],
    },
    {
        id: 'chat_bob',
        name: 'Bob Jones',
        avatarInitials: 'BJ',
        isGroup: false,
        unreadCount: 0,
        updatedAt: 'Yesterday',
        onlineStatus: 'Last seen yesterday at 18:20',
        messages: [
            {
                id: 'm7',
                chatId: 'chat_bob',
                sender: BOB_CONTACT,
                content: 'Let me know when the build is updated in bin/',
                timestamp: 'Yesterday 18:15',
                isOutgoing: false,
                status: 'read',
            },
            {
                id: 'm8',
                chatId: 'chat_bob',
                sender: ME_CONTACT,
                content: 'Will do! Stay tuned.',
                timestamp: 'Yesterday 18:20',
                isOutgoing: true,
                status: 'read',
            },
        ],
    },
    {
        id: 'chat_carol',
        name: 'Carol White',
        avatarInitials: 'CW',
        isGroup: false,
        unreadCount: 0,
        updatedAt: 'Jul 22',
        onlineStatus: 'Offline',
        messages: [
            {
                id: 'm9',
                chatId: 'chat_carol',
                sender: CAROL_CONTACT,
                content: 'Sent the mock design assets for GNOME top bar.',
                timestamp: 'Jul 22 14:05',
                isOutgoing: false,
                status: 'read',
            },
        ],
    },
];

// Calculate lastMessage dynamically
for (const chat of MOCK_CHATS) {
    if (chat.messages.length > 0) {
        chat.lastMessage = chat.messages[chat.messages.length - 1];
    }
}
