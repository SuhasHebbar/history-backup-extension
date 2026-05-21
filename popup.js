// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

const DEFAULT_UPLOAD_CONFIG = {
  uploadUrl: 'http://placeholder:9001/',
  uploadPeriodMinutes: 1
};

const STORAGE_KEY = 'historyUpload';

function getStatusElement() {
  return document.getElementById('uploadSettings_status');
}

function setSettingsStatus(message, type) {
  const statusElement = getStatusElement();
  statusElement.textContent = message;
  statusElement.className = type || '';
}

function getLastSuccessfulUploadTimeElement() {
  return document.getElementById('lastSuccessfulUploadTime_value');
}

function formatLastSuccessfulUploadTime(lastSuccessfulUploadTime) {
  if (!lastSuccessfulUploadTime) {
    return 'Never';
  }

  const date = new Date(lastSuccessfulUploadTime);
  if (Number.isNaN(date.getTime())) {
    return 'Never';
  }

  return date.toLocaleString();
}

function renderLastSuccessfulUploadTime(state) {
  const element = getLastSuccessfulUploadTimeElement();
  element.textContent = formatLastSuccessfulUploadTime(
    state && state.lastSuccessfulUploadTime
  );
}

function normalizePermissionOrigin(uploadUrl) {
  const url = new URL(uploadUrl);
  return `${url.protocol}//${url.hostname}/*`;
}

function requestPermission(origin) {
  return new Promise((resolve) => {
    chrome.permissions.request({ origins: [origin] }, (granted) => {
      if (chrome.runtime.lastError) {
        resolve(false);
        return;
      }

      resolve(granted);
    });
  });
}

function validateUploadUrl(uploadUrl) {
  if (!uploadUrl) {
    return null;
  }

  let url;
  try {
    url = new URL(uploadUrl);
  } catch (error) {
    throw new Error('Upload URL must be a valid URL.');
  }

  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error('Upload URL must start with http:// or https://.');
  }

  return url.href;
}

function validateUploadPeriod(uploadPeriodMinutes) {
  if (!uploadPeriodMinutes) {
    return null;
  }

  const value = Number(uploadPeriodMinutes);
  if (!Number.isFinite(value) || value <= 0) {
    throw new Error('Upload period must be a positive number.');
  }

  return value;
}

async function loadUploadSettings() {
  const data = await chrome.storage.local.get(STORAGE_KEY);
  const state = data[STORAGE_KEY] || {};

  const uploadUrlInput = document.getElementById('uploadUrl_input');
  const uploadPeriodInput = document.getElementById(
    'uploadPeriodMinutes_input'
  );

  uploadUrlInput.value =
    state.uploadUrl && state.uploadUrl !== DEFAULT_UPLOAD_CONFIG.uploadUrl
      ? state.uploadUrl
      : '';
  uploadPeriodInput.value =
    state.uploadPeriodMinutes &&
    state.uploadPeriodMinutes !== DEFAULT_UPLOAD_CONFIG.uploadPeriodMinutes
      ? String(state.uploadPeriodMinutes)
      : '';

  renderLastSuccessfulUploadTime(state);
}

async function saveUploadSettings(event) {
  event.preventDefault();
  setSettingsStatus('', '');

  const uploadUrlInput = document.getElementById('uploadUrl_input');
  const uploadPeriodInput = document.getElementById(
    'uploadPeriodMinutes_input'
  );

  try {
    const uploadUrl = validateUploadUrl(uploadUrlInput.value.trim());
    const uploadPeriodMinutes = validateUploadPeriod(
      uploadPeriodInput.value.trim()
    );

    if (uploadUrl) {
      const origin = normalizePermissionOrigin(uploadUrl);
      const granted = await requestPermission(origin);

      if (!granted) {
        throw new Error('Permission is required to upload to that URL.');
      }
    }

    const data = await chrome.storage.local.get(STORAGE_KEY);
    const nextState = {
      ...(data[STORAGE_KEY] || {})
    };

    if (uploadUrl) {
      nextState.uploadUrl = uploadUrl;
    } else {
      delete nextState.uploadUrl;
    }

    if (uploadPeriodMinutes) {
      nextState.uploadPeriodMinutes = uploadPeriodMinutes;
    } else {
      delete nextState.uploadPeriodMinutes;
    }

    await chrome.storage.local.set({
      [STORAGE_KEY]: nextState
    });

    setSettingsStatus('Saved.', 'success');
  } catch (error) {
    setSettingsStatus(error.message || String(error), 'error');
  }
}

function initializeUploadSettings() {
  const form = document.getElementById('uploadSettings_form');
  form.addEventListener('submit', saveUploadSettings);

  loadUploadSettings().catch((error) => {
    setSettingsStatus(error.message || String(error), 'error');
  });

  chrome.storage.onChanged.addListener((changes, areaName) => {
    if (areaName !== 'local' || !changes[STORAGE_KEY]) {
      return;
    }

    renderLastSuccessfulUploadTime(changes[STORAGE_KEY].newValue || {});
  });
}

document.addEventListener('DOMContentLoaded', function () {
  initializeUploadSettings();
});
