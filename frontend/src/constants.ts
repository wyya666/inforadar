export const ApiBasePath = '/api/v1'

// Keep these values aligned with internal/constants/response.go.
export const ApiErrorCode = {
  Network: -1,
  System: 50000,
  RequestTooLarge: 41300,
  RadarUnconfigured: 40010,
  RadarBusy: 40910,
  RadarCooldown: 42910,
  ProviderUnavailable: 50310,
} as const
