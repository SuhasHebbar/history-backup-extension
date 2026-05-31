importScripts('shared-config.js');

const { DEFAULT_UPLOAD_CONFIG, STORAGE_KEY, ensureDeviceName } =
  HistoryUploadConfig;
const ALARM_NAME = 'uploadHistory';
const MAX_HISTORY_RESULTS = 86400 * 90 * 10;
const UPLOAD_MODE_ALL = 'all';
const UPLOAD_MODE_INCREMENTAL = 'incremental';

let uploadInProgress = false;

async function getUploadState() {
  const data = await chrome.storage.local.get(STORAGE_KEY);
  return {
    ...DEFAULT_UPLOAD_CONFIG,
    ...(data[STORAGE_KEY] || {})
  };
}

async function saveUploadState(updates) {
  const data = await chrome.storage.local.get(STORAGE_KEY);
  const currentState = data[STORAGE_KEY] || {};

  await chrome.storage.local.set({
    [STORAGE_KEY]: {
      ...currentState,
      ...updates
    }
  });
}

async function ensureUploadAlarm() {
  await ensureDeviceName();

  const state = await getUploadState();

  await chrome.alarms.create(ALARM_NAME, {
    periodInMinutes: state.uploadPeriodMinutes
  });
}

function getEffectiveUploadPeriod(state) {
  return (
    (state && state.uploadPeriodMinutes) ||
    DEFAULT_UPLOAD_CONFIG.uploadPeriodMinutes
  );
}

function serializeHistoryItem(item) {
  return {
    id: item.id,
    url: item.url,
    title: item.title,
    lastVisitTime: item.lastVisitTime,
    visitCount: item.visitCount,
    typedCount: item.typedCount
  };
}

function getRangeStartTime(state, mode) {
  if (mode === UPLOAD_MODE_ALL) {
    return 0;
  }

  return state.lastSuccessfulUploadTime || 0;
}

async function uploadHistory(options = {}) {
  if (uploadInProgress) {
    throw new Error('Upload already in progress.');
  }

  uploadInProgress = true;

  const rangeEndTime = Date.now();
  const mode = options.mode || UPLOAD_MODE_INCREMENTAL;

  try {
    const deviceName = await ensureDeviceName();
    const state = await getUploadState();

    if (!state.uploadUrl) {
      await saveUploadState({ lastError: 'No upload URL configured.' });
      return;
    }

    const rangeStartTime = getRangeStartTime(state, mode);

    await saveUploadState({
      lastAttemptTime: rangeEndTime,
      lastError: null
    });

    const historyItems = await chrome.history.search({
      text: '',
      startTime: rangeStartTime,
      endTime: rangeEndTime,
      maxResults: MAX_HISTORY_RESULTS
    });

    const items = historyItems.map(serializeHistoryItem);
    const uploadedAt = Date.now();

    if (items.length > 0) {
      const response = await fetch(state.uploadUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          uploadedAt,
          deviceName,
          rangeStartTime,
          rangeEndTime,
          items
        })
      });

      if (!response.ok) {
        throw new Error(`Upload failed with HTTP ${response.status}`);
      }
    }

    await saveUploadState({
      lastSuccessfulUploadTime: rangeEndTime,
      lastUploadedCount: items.length,
      lastError: null
    });

    return {
      uploadedCount: items.length,
      rangeStartTime,
      rangeEndTime
    };
  } catch (error) {
    await saveUploadState({
      lastError: error.message || String(error)
    });

    throw error;
  } finally {
    uploadInProgress = false;
  }
}

chrome.runtime.onInstalled.addListener(() => {
  ensureUploadAlarm().catch((error) => {
    console.error('Failed to initialize history upload alarm:', error);
  });
});

chrome.runtime.onStartup.addListener(() => {
  ensureUploadAlarm().catch((error) => {
    console.error('Failed to initialize history upload alarm:', error);
  });
});

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === ALARM_NAME) {
    uploadHistory({ mode: UPLOAD_MODE_INCREMENTAL }).catch((error) => {
      console.error('Failed to upload history:', error);
    });
  }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (!message || message.type !== 'uploadHistory') {
    return false;
  }

  const mode =
    message.mode === UPLOAD_MODE_ALL ? UPLOAD_MODE_ALL : UPLOAD_MODE_INCREMENTAL;

  uploadHistory({ mode })
    .then((result) => {
      sendResponse({
        ok: true,
        uploadedCount: result.uploadedCount,
        rangeStartTime: result.rangeStartTime,
        rangeEndTime: result.rangeEndTime
      });
    })
    .catch((error) => {
      sendResponse({
        ok: false,
        error: error.message || String(error)
      });
    });

  return true;
});

chrome.storage.onChanged.addListener((changes, areaName) => {
  if (areaName !== 'local' || !changes[STORAGE_KEY]) {
    return;
  }

  const oldPeriod = getEffectiveUploadPeriod(changes[STORAGE_KEY].oldValue);
  const newPeriod = getEffectiveUploadPeriod(changes[STORAGE_KEY].newValue);

  if (oldPeriod !== newPeriod) {
    ensureUploadAlarm().catch((error) => {
      console.error('Failed to update history upload alarm:', error);
    });
  }
});
