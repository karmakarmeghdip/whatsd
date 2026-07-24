import GObject from 'gi://GObject?version=2.0';
import Gdk from 'gi://Gdk?version=4.0';
import Gtk from 'gi://Gtk?version=4.0';
import Gio from 'gi://Gio?version=2.0';
import Adw from 'gi://Adw?version=1';
import Template from './window.blp';
import { MOCK_CHATS, ME_CONTACT } from './mock-data.js';
import { ChatRow } from './components/chat-row.js';
import { MessageBubble } from './components/message-bubble.js';
import { showAboutDialog } from './components/about-dialog.js';
import { showQRDialog } from './components/qr-dialog.js';
import type { Chat, Message } from './types.js';

export class MainWindow extends Adw.ApplicationWindow {
    declare private _menuButton: Gtk.MenuButton;
    declare private _searchEntry: Gtk.SearchEntry;
    declare private _sidebarList: Gtk.ListBox;
    declare private _chatTitle: Adw.WindowTitle;
    declare private _infoButton: Gtk.Button;
    declare private _contentStack: Gtk.Stack;
    declare private _emptyPage: Adw.StatusPage;
    declare private _chatViewBox: Gtk.Box;
    declare private _messageScroll: Gtk.ScrolledWindow;
    declare private _messageList: Gtk.Box;
    declare private _messageEntry: Gtk.Entry;
    declare private _sendButton: Gtk.Button;
    declare private _attachButton: Gtk.Button;

    private _activeChat: Chat | null = null;
    private _chats: Chat[] = MOCK_CHATS;

    static {
        GObject.registerClass(
            {
                GTypeName: 'MainWindow',
                Template,
                InternalChildren: [
                    'menuButton',
                    'searchEntry',
                    'sidebarList',
                    'chatTitle',
                    'infoButton',
                    'contentStack',
                    'emptyPage',
                    'chatViewBox',
                    'messageScroll',
                    'messageList',
                    'messageEntry',
                    'sendButton',
                    'attachButton',
                ],
            },
            this,
        );
    }

    constructor(application: Adw.Application) {
        super({ application });

        this._loadCss();
        this._setupMenuActions();
        this._setupEvents();
        this._populateSidebar();
    }

    private _loadCss(): void {
        const cssString = `
            .chat-title-text { font-weight: bold; font-size: 14px; }
            .chat-preview-text { font-size: 12px; opacity: 0.7; }
            .chat-timestamp-text { font-size: 11px; opacity: 0.6; }
            .chat-unread-badge { background-color: @accent_bg_color; color: @accent_fg_color; border-radius: 9999px; font-weight: bold; font-size: 11px; padding: 2px 7px; }
            .message-bubble-outgoing { background-color: alpha(@accent_color, 0.18); border-radius: 14px 14px 2px 14px; padding: 8px 12px; margin-left: 48px; margin-right: 4px; }
            .message-bubble-incoming { background-color: alpha(@card_shade_color, 0.6); border-radius: 14px 14px 14px 2px; padding: 8px 12px; margin-left: 4px; margin-right: 48px; }
            .message-sender-name { font-weight: bold; font-size: 11px; color: @accent_color; margin-bottom: 2px; }
            .message-text { font-size: 13px; }
            .message-time-stamp { font-size: 10px; opacity: 0.55; margin-top: 4px; }
        `;
        const provider = new Gtk.CssProvider();
        provider.load_from_data(cssString, -1);
        const display = Gdk.Display.get_default();
        if (display) {
            Gtk.StyleContext.add_provider_for_display(
                display,
                provider,
                Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
            );
        }
    }

    private _setupMenuActions(): void {
        const menu = new Gio.Menu();
        menu.append('Pair Device (QR Code)', 'win.qr-status');
        menu.append('About WhatsApp Desktop', 'win.about');
        this._menuButton.set_menu_model(menu);

        const actionGroup = new Gio.SimpleActionGroup();

        const aboutAction = new Gio.SimpleAction({ name: 'about' });
        aboutAction.connect('activate', () => {
            showAboutDialog(this);
        });
        actionGroup.add_action(aboutAction);

        const qrAction = new Gio.SimpleAction({ name: 'qr-status' });
        qrAction.connect('activate', () => {
            showQRDialog(this);
        });
        actionGroup.add_action(qrAction);

        this.insert_action_group('win', actionGroup);
    }

    private _setupEvents(): void {
        this._sidebarList.connect('row-selected', (_list: Gtk.ListBox, row: Gtk.ListBoxRow | null) => {
            if (row instanceof ChatRow) {
                this.selectChat(row.chat);
            }
        });

        this._searchEntry.connect('search-changed', (entry: Gtk.SearchEntry) => {
            const query = entry.get_text().toLowerCase().trim();
            this._filterSidebar(query);
        });

        this._sendButton.connect('clicked', () => {
            this._sendMessage();
        });

        this._messageEntry.connect('activate', () => {
            this._sendMessage();
        });

        this._infoButton.connect('clicked', () => {
            if (this._activeChat) {
                showQRDialog(this);
            }
        });
    }

    private _populateSidebar(): void {
        let child = this._sidebarList.get_first_child();
        while (child) {
            const next = child.get_next_sibling();
            this._sidebarList.remove(child);
            child = next;
        }

        for (const chat of this._chats) {
            const row = new ChatRow(chat);
            this._sidebarList.append(row);
        }
    }

    private _filterSidebar(query: string): void {
        let child = this._sidebarList.get_first_child();
        while (child) {
            if (child instanceof ChatRow) {
                const matches = query === '' || child.chat.name.toLowerCase().includes(query);
                child.set_visible(matches);
            }
            child = child.get_next_sibling();
        }
    }

    public selectChat(chat: Chat): void {
        this._activeChat = chat;
        chat.unreadCount = 0;

        this._chatTitle.set_title(chat.name);
        this._chatTitle.set_subtitle(chat.onlineStatus || '');
        this._infoButton.set_sensitive(true);

        this._contentStack.set_visible_child(this._chatViewBox);

        this._renderMessages(chat);
        this._populateSidebar();
    }

    private _renderMessages(chat: Chat): void {
        let child = this._messageList.get_first_child();
        while (child) {
            const next = child.get_next_sibling();
            this._messageList.remove(child);
            child = next;
        }

        for (const msg of chat.messages) {
            const bubble = new MessageBubble(msg);
            this._messageList.append(bubble);
        }

        // Scroll to bottom
        const adj = this._messageScroll.get_vadjustment();
        if (adj) {
            adj.set_value(adj.get_upper() - adj.get_page_size());
        }
    }

    private _sendMessage(): void {
        if (!this._activeChat) return;

        const text = this._messageEntry.get_text().trim();
        if (!text) return;

        const now = new Date();
        const timeStr = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}`;

        const newMsg: Message = {
            id: `msg_${Date.now()}`,
            chatId: this._activeChat.id,
            sender: ME_CONTACT,
            content: text,
            timestamp: timeStr,
            isOutgoing: true,
            status: 'sent',
        };

        this._activeChat.messages.push(newMsg);
        this._activeChat.lastMessage = newMsg;
        this._activeChat.updatedAt = timeStr;

        this._messageEntry.set_text('');
        this._renderMessages(this._activeChat);
        this._populateSidebar();
    }
}
