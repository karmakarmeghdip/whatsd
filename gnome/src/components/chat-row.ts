import GObject from 'gi://GObject?version=2.0';
import Adw from 'gi://Adw?version=1';
import Gtk from 'gi://Gtk?version=4.0';
import type { Chat } from '../types.js';

export class ChatRow extends Gtk.ListBoxRow {
    private _chat!: Chat;

    static {
        GObject.registerClass({ GTypeName: 'ChatRow' }, this);
    }

    constructor(chat: Chat) {
        super();
        this._chat = chat;

        const mainBox = new Gtk.Box({
            orientation: Gtk.Orientation.HORIZONTAL,
            spacing: 12,
            margin_top: 8,
            margin_bottom: 8,
            margin_start: 10,
            margin_end: 10,
        });

        // Avatar
        const avatar = new Adw.Avatar({
            size: 40,
            text: chat.name,
            show_initials: true,
        });
        mainBox.append(avatar);

        // Text details box (vertical)
        const detailsBox = new Gtk.Box({
            orientation: Gtk.Orientation.VERTICAL,
            spacing: 2,
            hexpand: true,
            valign: Gtk.Align.CENTER,
        });

        // Top line: Name + Time
        const topRow = new Gtk.Box({
            orientation: Gtk.Orientation.HORIZONTAL,
            spacing: 6,
        });

        const nameLabel = new Gtk.Label({
            label: chat.name,
            xalign: 0,
            hexpand: true,
            ellipsize: 3, // Pango.EllipsizeMode.END
        });
        nameLabel.add_css_class('chat-title-text');

        const timeLabel = new Gtk.Label({
            label: chat.updatedAt,
            xalign: 1,
        });
        timeLabel.add_css_class('chat-timestamp-text');

        topRow.append(nameLabel);
        topRow.append(timeLabel);
        detailsBox.append(topRow);

        // Bottom line: Last message preview + Unread badge
        const bottomRow = new Gtk.Box({
            orientation: Gtk.Orientation.HORIZONTAL,
            spacing: 6,
        });

        const previewText = chat.lastMessage
            ? chat.lastMessage.content
            : 'No messages yet';

        const previewLabel = new Gtk.Label({
            label: previewText,
            xalign: 0,
            hexpand: true,
            ellipsize: 3, // Pango.EllipsizeMode.END
        });
        previewLabel.add_css_class('chat-preview-text');

        bottomRow.append(previewLabel);

        if (chat.unreadCount > 0) {
            const badgeLabel = new Gtk.Label({
                label: chat.unreadCount.toString(),
            });
            badgeLabel.add_css_class('chat-unread-badge');
            bottomRow.append(badgeLabel);
        }

        detailsBox.append(bottomRow);
        mainBox.append(detailsBox);

        this.set_child(mainBox);
    }

    public get chat(): Chat {
        return this._chat;
    }
}
