import Adw from 'gi://Adw?version=1';
import Gtk from 'gi://Gtk?version=4.0';

export function showQRDialog(parentWindow: Gtk.Window): void {
    const dialog = new Adw.Dialog({
        title: 'Pair Device — whatsd Daemon',
        content_width: 440,
        content_height: 520,
    });

    const toolbarView = new Adw.ToolbarView();
    const headerBar = new Adw.HeaderBar();
    toolbarView.add_top_bar(headerBar);

    const statusPage = new Adw.StatusPage({
        icon_name: 'chat-message-new-symbolic',
        title: 'Link WhatsApp Device',
        description:
            'Start the whatsd daemon in your terminal or via systemd. When QR code authentication is required, scan the QR code printed by the daemon or displayed here.',
        vexpand: true,
    });

    const group = new Adw.PreferencesGroup({
        title: 'IPC Service Connection',
        margin_top: 12,
        margin_bottom: 12,
        margin_start: 16,
        margin_end: 16,
    });

    const socketRow = new Adw.ActionRow({
        title: 'Unix Socket IPC Status',
        subtitle: 'Listening on /tmp/whatsd.sock',
        icon_name: 'dialog-information-symbolic',
    });

    group.add(socketRow);
    statusPage.set_child(group);

    toolbarView.set_content(statusPage);
    dialog.set_child(toolbarView);
    dialog.present(parentWindow);
}
