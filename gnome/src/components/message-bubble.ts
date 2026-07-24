import GObject from 'gi://GObject?version=2.0';
import Gtk from 'gi://Gtk?version=4.0';
import type { Message } from '../types.js';

export class MessageBubble extends Gtk.Box {
    private _message!: Message;

    static {
        GObject.registerClass({ GTypeName: 'MessageBubble' }, this);
    }

    constructor(message: Message) {
        super({
            orientation: Gtk.Orientation.VERTICAL,
            spacing: 2,
        });
        this._message = message;

        const isOutgoing = message.isOutgoing;

        const bubbleBox = new Gtk.Box({
            orientation: Gtk.Orientation.VERTICAL,
            spacing: 4,
            halign: isOutgoing ? Gtk.Align.END : Gtk.Align.START,
        });

        if (isOutgoing) {
            bubbleBox.add_css_class('message-bubble-outgoing');
        } else {
            bubbleBox.add_css_class('message-bubble-incoming');
        }

        // Sender name if group message incoming
        if (!isOutgoing && message.sender.name) {
            const senderLabel = new Gtk.Label({
                label: message.sender.name,
                xalign: 0,
            });
            senderLabel.add_css_class('message-sender-name');
            bubbleBox.append(senderLabel);
        }

        // Message text body
        const contentLabel = new Gtk.Label({
            label: message.content,
            xalign: 0,
            wrap: true,
            wrap_mode: 2, // Pango.WrapMode.WORD_CHAR
            selectable: true,
        });
        contentLabel.add_css_class('message-text');
        bubbleBox.append(contentLabel);

        // Footer info: Timestamp + Status indicator icon for outgoing
        const footerBox = new Gtk.Box({
            orientation: Gtk.Orientation.HORIZONTAL,
            spacing: 4,
            halign: isOutgoing ? Gtk.Align.END : Gtk.Align.START,
        });

        const timeLabel = new Gtk.Label({
            label: message.timestamp,
        });
        timeLabel.add_css_class('message-time-stamp');
        footerBox.append(timeLabel);

        if (isOutgoing) {
            const statusIcon = new Gtk.Image({
                icon_name: message.status === 'read' ? 'error-correct-symbolic' : 'emblem-synchronizing-symbolic',
                pixel_size: 12,
            });
            statusIcon.add_css_class('message-time-stamp');
            footerBox.append(statusIcon);
        }

        bubbleBox.append(footerBox);
        this.append(bubbleBox);
    }

    public get message(): Message {
        return this._message;
    }
}
