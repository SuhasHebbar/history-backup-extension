const DEFAULT_CONFIG = {
  uploadUrl: 'http://placeholder:9001/',
  uploadPeriodMinutes: 1
};

const STORAGE_KEY = 'historyUpload';
const ALARM_NAME = 'uploadHistory';
const MAX_HISTORY_RESULTS = 86400 * 90 * 10;

let uploadInProgress = false;

async function getUploadState() {
  const data = await chrome.storage.local.get(STORAGE_KEY);
  return {
    ...DEFAULT_CONFIG,
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
  const state = await getUploadState();

  await chrome.alarms.create(ALARM_NAME, {
    periodInMinutes: state.uploadPeriodMinutes
  });
}

function getEffectiveUploadPeriod(state) {
  return (state && state.uploadPeriodMinutes) || DEFAULT_CONFIG.uploadPeriodMinutes;
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

async function uploadHistory() {
  if (uploadInProgress) {
    return;
  }

  uploadInProgress = true;

  const rangeEndTime = Date.now();

  try {
    const state = await getUploadState();
    const rangeStartTime = state.lastSuccessfulUploadTime || 0;

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
  } catch (error) {
    await saveUploadState({
      lastError: error.message || String(error)
    });
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
    uploadHistory().catch((error) => {
      console.error('Failed to upload history:', error);
    });
  }
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
