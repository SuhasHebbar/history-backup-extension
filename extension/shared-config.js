const HistoryUploadConfig = (() => {
  const DEFAULT_UPLOAD_CONFIG = {
    uploadPeriodMinutes: 60
  };

  const STORAGE_KEY = 'historyUpload';

  const DEVICE_ADJECTIVES = [
    'able',
    'active',
    'alert',
    'amber',
    'amused',
    'brave',
    'bright',
    'calm',
    'careful',
    'charming',
    'cheerful',
    'clever',
    'cool',
    'cosmic',
    'crisp',
    'curious',
    'dapper',
    'daring',
    'deft',
    'eager',
    'early',
    'easy',
    'electric',
    'elegant',
    'fair',
    'fast',
    'fine',
    'fleet',
    'fresh',
    'gentle',
    'glad',
    'golden',
    'grand',
    'happy',
    'hardy',
    'honest',
    'humble',
    'ideal',
    'jolly',
    'keen',
    'kind',
    'lively',
    'loyal',
    'lucky',
    'merry',
    'mighty',
    'modern',
    'neat',
    'nimble',
    'noble',
    'novel',
    'patient',
    'peaceful',
    'playful',
    'polite',
    'proud',
    'quick',
    'quiet',
    'radiant',
    'ready',
    'regal',
    'robust',
    'sharp',
    'shiny',
    'simple',
    'sincere',
    'smart',
    'snappy',
    'steady',
    'sunny',
    'swift',
    'tidy',
    'trusty',
    'upbeat',
    'urban',
    'vivid',
    'warm',
    'witty',
    'zesty',
    'agile',
    'balanced',
    'bold',
    'candid',
    'classic',
    'clean',
    'crafty',
    'dreamy',
    'earnest',
    'fancy',
    'fearless',
    'focused',
    'friendly',
    'graceful',
    'helpful',
    'hopeful',
    'inventive',
    'joyful',
    'lucid',
    'plucky',
    'stellar'
  ];

  const DEVICE_NAMES = [
    'alex',
    'avery',
    'bailey',
    'blair',
    'casey',
    'chris',
    'dana',
    'devon',
    'drew',
    'elliot',
    'emery',
    'finley',
    'frankie',
    'gray',
    'harper',
    'hayden',
    'jamie',
    'jordan',
    'jules',
    'kai',
    'kendall',
    'lane',
    'logan',
    'morgan',
    'parker',
    'peyton',
    'quinn',
    'reese',
    'riley',
    'river',
    'robin',
    'rowan',
    'sage',
    'sam',
    'sawyer',
    'shawn',
    'skyler',
    'taylor',
    'terry',
    'tracy',
    'val',
    'winter',
    'addison',
    'adrien',
    'aiden',
    'andy',
    'arden',
    'ari',
    'ash',
    'aubrey',
    'beck',
    'bellamy',
    'brennan',
    'brook',
    'cam',
    'cameron',
    'carson',
    'charlie',
    'dakota',
    'dallas',
    'denver',
    'eden',
    'ellis',
    'emerson',
    'erin',
    'francis',
    'gale',
    'greer',
    'hadley',
    'hollis',
    'hunter',
    'indy',
    'jaden',
    'jay',
    'jesse',
    'jo',
    'kelly',
    'kit',
    'lennon',
    'linden',
    'marley',
    'micah',
    'monroe',
    'nico',
    'noel',
    'oakley',
    'phoenix',
    'remy',
    'rory',
    'shiloh',
    'sidney',
    'spencer',
    'stevie',
    'sutton',
    'teagan',
    'toby',
    'wren',
    'yael',
    'zion',
    'max'
  ];

  function getRandomIndex(length) {
    const values = new Uint32Array(1);
    crypto.getRandomValues(values);
    return values[0] % length;
  }

  function getRandomItem(items) {
    return items[getRandomIndex(items.length)];
  }

  function generateDeviceName() {
    return `${getRandomItem(DEVICE_ADJECTIVES)}-${getRandomItem(DEVICE_NAMES)}`;
  }

  async function ensureDeviceName() {
    const data = await chrome.storage.local.get(STORAGE_KEY);
    const state = data[STORAGE_KEY] || {};

    if (state.deviceName) {
      return state.deviceName;
    }

    const deviceName = generateDeviceName();
    await chrome.storage.local.set({
      [STORAGE_KEY]: {
        ...state,
        deviceName
      }
    });

    return deviceName;
  }

  return {
    DEFAULT_UPLOAD_CONFIG,
    STORAGE_KEY,
    DEVICE_ADJECTIVES,
    DEVICE_NAMES,
    generateDeviceName,
    ensureDeviceName
  };
})();
