use whatsapp_rust::pair_code::PairCodeOptions;

use crate::types::PairCodePayload;

use super::error::SessionError;

pub(super) fn pair_code_options(payload: PairCodePayload) -> Result<PairCodeOptions, SessionError> {
    Ok(PairCodeOptions {
        phone_number: normalize_phone_number(&payload.phone_number)?,
        custom_code: normalize_custom_code(payload.custom_code)?,
        ..Default::default()
    })
}

fn normalize_phone_number(phone_number: &str) -> Result<String, SessionError> {
    let phone_number = phone_number
        .chars()
        .filter(|c| c.is_ascii_digit())
        .collect::<String>();

    if phone_number.is_empty() {
        return Err(SessionError::InvalidRequest(
            "phone_number is required".to_owned(),
        ));
    }

    if phone_number.len() < 7 {
        return Err(SessionError::InvalidRequest(
            "phone_number must include country code and at least 7 digits".to_owned(),
        ));
    }

    if phone_number.starts_with('0') {
        return Err(SessionError::InvalidRequest(
            "phone_number must be international and must not start with 0".to_owned(),
        ));
    }

    Ok(phone_number)
}

fn normalize_custom_code(custom_code: Option<String>) -> Result<Option<String>, SessionError> {
    let Some(custom_code) = custom_code else {
        return Ok(None);
    };

    let custom_code = custom_code.to_ascii_uppercase();
    if custom_code.len() != 8
        || !custom_code
            .bytes()
            .all(|byte| b"123456789ABCDEFGHJKLMNPQRSTVWXYZ".contains(&byte))
    {
        return Err(SessionError::InvalidRequest(
            "custom_code must be 8 Crockford characters".to_owned(),
        ));
    }

    Ok(Some(custom_code))
}
