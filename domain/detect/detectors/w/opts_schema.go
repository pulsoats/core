package w

import "encoding/json"

var optsSchema = json.RawMessage(`{
  "type": "object",
  "required": [
    "LocalMinsDeviation",
    "MinMaxDeviation",
    "TakeProfitRatio",
    "VolumeSpikeMultiplier",
    "VolumeCVThreshold",
    "StopLossRatio",
    "BarsForBuy",
    "BarsForSell",
    "WindowSize"
  ],
  "properties": {
    "LocalMinsDeviation": {
      "type": "integer",
      "minimum": 0,
      "description": "PPM допустимого отличия между минимумами"
    },
    "MinMaxDeviation": {
      "type": "integer",
      "minimum": 0,
      "description": "PPM амплитуды между минимумом и максимумом"
    },
    "TakeProfitRatio": {
      "type": "integer",
      "minimum": 0,
      "description": "PPM множитель цены take-profit"
    },
    "VolumeSpikeMultiplier": {
      "type": "integer",
      "minimum": 0
    },
    "VolumeCVThreshold": {
      "type": "integer",
      "minimum": 0
    },
    "StopLossRatio": {
      "type": "integer",
      "minimum": 0,
      "description": "PPM множитель цены stop-loss"
    },
    "BarsForBuy": {
      "type": "integer",
      "minimum": 1,
      "description": "Количество свечей для покупки"
    },
    "BarsForSell": {
      "type": "integer",
      "minimum": 1,
      "description": "Количество свечей для продажи"
    },
    "WindowSize": {
      "type": "integer",
      "minimum": 5,
      "description": "Размер окна"
    }
  },
  "additionalProperties": false
}`)
