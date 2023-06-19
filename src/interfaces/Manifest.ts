export interface Manifest {
  allowCache: boolean;
  discontinuityStarts: [];
  segments: { duration: number; uri: string; timeline: number }[];
  mediaGroups: {
    AUDIO: null;
    VIDEO: {
      [key: string]: {
        [key: string]: { default: boolean; autoselect: boolean };
      };
    };
    "CLOSED-CAPTIONS": null;
    SUBTITLES: null;
  };
  playlists: {
    attributes: {
      "FRAME-RATE": number;
      VIDEO: string;
      RESOLUTION: { width: number; height: number };
      CODECS: string;
      BANDWIDTH: number;
    };
    uri: string;
    timeline: number;
  }[];
}
