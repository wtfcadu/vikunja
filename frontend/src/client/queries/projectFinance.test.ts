import {describe, expect, it} from 'vitest'

import {formatMoney, minorToInput, parseMoney, totalsByCurrency} from './projectFinance'
import type {ProjectOverviewResponse} from './projectFinance'

describe('parseMoney', () => {
	it('reads plain and dotted amounts', () => {
		expect(parseMoney('1250')).toBe(125000)
		expect(parseMoney('1,250.50')).toBe(125050)
		expect(parseMoney('£ 99.99')).toBe(9999)
	})

	it('reads a comma decimal separator', () => {
		expect(parseMoney('1250,5')).toBe(125050)
		expect(parseMoney('1.250,50')).toBe(125050)
	})

	it('treats nonsense and negatives as zero', () => {
		expect(parseMoney('')).toBe(0)
		expect(parseMoney('abc')).toBe(0)
		expect(parseMoney('-20')).toBe(0)
	})
})

describe('formatMoney', () => {
	it('formats with a currency symbol', () => {
		expect(formatMoney(125050, 'GBP', 'en-GB')).toBe('£1,250.50')
	})

	it('falls back to a plain number without a currency', () => {
		expect(formatMoney(125050, '', 'en-GB')).toBe('1,250.50')
	})

	it('round-trips through the input format', () => {
		expect(parseMoney(minorToInput(98765))).toBe(98765)
	})
})

describe('totalsByCurrency', () => {
	function row(currency: string, fee: number, paid: number): ProjectOverviewResponse {
		return {
			id: 1, title: 'p', hex_color: '', is_archived: false, parent_project_id: 0,
			client: '', currency, fee, paid, owed: Math.max(fee - paid, 0), payments: 0,
			open_tasks: 0, done_tasks: 0, my_open_tasks: 0, workers: [],
		}
	}

	it('keeps currencies apart and skips projects without money', () => {
		const totals = totalsByCurrency([
			row('GBP', 1000, 400),
			row('GBP', 500, 500),
			row('EUR', 2000, 0),
			row('', 0, 0),
		])
		expect(totals).toEqual([
			{currency: 'EUR', fee: 2000, paid: 0, owed: 2000, projects: 1},
			{currency: 'GBP', fee: 1500, paid: 900, owed: 600, projects: 2},
		])
	})
})
