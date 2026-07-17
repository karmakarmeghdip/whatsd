use crate::types::{
    ContactGetPayload, ContactPayload, ContactProfilePicture, ContactProfilePicturePayload,
    ContactProfilePictureResultPayload,
};

use super::{error::SessionError, manager::SessionManager, messages::parse_full_jid_field};

impl SessionManager {
    pub async fn get_contact(
        &self,
        payload: ContactGetPayload,
    ) -> Result<Option<ContactPayload>, SessionError> {
        let jid = parse_full_jid_field(&payload.jid, "jid")?;
        let client = self.client_for_connected().await?;
        let requested_jid = jid.clone();
        let handle = tokio::runtime::Handle::current();
        // Upstream contact feature futures borrow a short-lived handle and are not Send.
        let mut info = tokio::task::spawn_blocking(move || {
            handle.block_on(async move {
                client
                    .contacts()
                    .get_user_info(std::slice::from_ref(&jid))
                    .await
            })
        })
        .await
        .map_err(|error| SessionError::ContactFailed(error.into()))?
        .map_err(SessionError::ContactFailed)?;

        Ok(info.remove(&requested_jid).map(|info| ContactPayload {
            jid: info.jid.to_string(),
            lid_jid: info.lid.map(|jid| jid.to_string()),
            status: info.status,
            picture_id: info.picture_id,
            is_business: info.is_business,
        }))
    }

    pub async fn get_contact_profile_picture(
        &self,
        payload: ContactProfilePicturePayload,
    ) -> Result<ContactProfilePictureResultPayload, SessionError> {
        let jid = parse_full_jid_field(&payload.jid, "jid")?;
        let client = self.client_for_connected().await?;
        let preview = payload.preview;
        let requested_jid = jid.clone();
        let handle = tokio::runtime::Handle::current();
        // Upstream contact feature futures borrow a short-lived handle and are not Send.
        let picture = tokio::task::spawn_blocking(move || {
            handle
                .block_on(async move { client.contacts().get_profile_picture(&jid, preview).await })
        })
        .await
        .map_err(|error| SessionError::ContactFailed(error.into()))?
        .map_err(SessionError::ContactFailed)?
        .map(|picture| ContactProfilePicture {
            id: picture.id,
            url: picture.url,
            direct_path: picture.direct_path,
            hash: picture.hash,
        });

        Ok(ContactProfilePictureResultPayload {
            jid: requested_jid.to_string(),
            picture,
        })
    }
}
