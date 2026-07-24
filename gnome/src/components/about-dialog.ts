import Adw from 'gi://Adw?version=1';
import Gtk from 'gi://Gtk?version=4.0';

export function showAboutDialog(parentWindow: Gtk.Window): void {
    const dialog = new Adw.AboutDialog({
        application_name: 'WhatsApp Desktop',
        application_icon: 'chat-message-new-symbolic',
        developer_name: 'Meghdip Karmakar',
        version: '0.1.0',
        comments: 'Native GTK4 / LibAdwaita client for the whatsd WhatsApp daemon.',
        website: 'https://github.com/karmakarmeghdip/whatsd',
        issue_url: 'https://github.com/karmakarmeghdip/whatsd/issues',
        license_type: Gtk.License.MIT_X11,
        copyright: '© 2026 Meghdip Karmakar',
    });

    dialog.present(parentWindow);
}
