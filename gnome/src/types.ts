export type ConnectionState = 'connected' | 'connecting' | 'disconnected' | 'qr_required';

export interface Contact {
    id: string;
    name: string;
    phoneNumber?: string;
    avatarInitials: string;
    isGroup: boolean;
}

export interface Message {
    id: string;
    chatId: string;
    sender: Contact;
    content: string;
    timestamp: string;
    isOutgoing: boolean;
    status: 'pending' | 'sent' | 'delivered' | 'read';
}

export interface Chat {
    id: string;
    name: string;
    avatarInitials: string;
    isGroup: boolean;
    unreadCount: number;
    lastMessage?: Message;
    updatedAt: string;
    onlineStatus?: string;
    messages: Message[];
}
