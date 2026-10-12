<template>
	<CreateEdit
		:title="project.title ? $t('projectFinance.modalTitle', {project: project.title}) : $t('projectFinance.money')"
		primary-icon=""
		:primary-label="$t('misc.save')"
		:primary-disabled="!financeLoaded || saveFinance.isPending.value"
		:loading="saveFinance.isPending.value"
		wide
		@primary="saveDetails"
	>
		<Message
			v-if="loadError"
			variant="danger"
		>
			{{ $t('projectFinance.ownerOnly') }}
		</Message>

		<template v-else>
			<div class="columns">
				<div class="column is-6">
					<FormField
						id="finance-client"
						v-model="draft.client"
						:label="$t('projectFinance.client')"
						:placeholder="$t('projectFinance.clientPlaceholder')"
						:disabled="!financeLoaded"
						list="finance-known-clients"
						type="text"
						maxlength="250"
						@keyup.enter="saveDetails"
					/>
					<datalist id="finance-known-clients">
						<option
							v-for="client in knownClients"
							:key="client"
							:value="client"
						/>
					</datalist>
				</div>
				<div class="column is-2">
					<FormField
						id="finance-currency"
						v-model="draft.currency"
						:label="$t('projectFinance.currency')"
						:disabled="!financeLoaded"
						list="finance-currencies"
						type="text"
						maxlength="3"
						placeholder="GBP"
						@keyup.enter="saveDetails"
					/>
					<datalist id="finance-currencies">
						<option
							v-for="code in COMMON_CURRENCIES"
							:key="code"
							:value="code"
						/>
					</datalist>
				</div>
				<div class="column is-4">
					<FormField
						id="finance-fee"
						v-model="draft.fee"
						:label="$t('projectFinance.fee')"
						:disabled="!financeLoaded"
						type="text"
						inputmode="decimal"
						placeholder="0.00"
						@keyup.enter="saveDetails"
					/>
				</div>
			</div>

			<div class="finance-summary">
				<div>
					<span class="finance-summary__label">{{ $t('projectFinance.fee') }}</span>
					<span class="finance-summary__value">{{ formatMoney(feeMinor, currency) }}</span>
				</div>
				<div>
					<span class="finance-summary__label">{{ $t('projectFinance.paid') }}</span>
					<span class="finance-summary__value has-text-success">{{ formatMoney(paidMinor, currency) }}</span>
				</div>
				<div>
					<span class="finance-summary__label">{{ $t('projectFinance.owed') }}</span>
					<span
						class="finance-summary__value"
						:class="{'has-text-danger': owedMinor > 0}"
					>{{ formatMoney(owedMinor, currency) }}</span>
				</div>
				<progress
					class="progress is-success finance-summary__bar"
					:value="paidMinor"
					:max="Math.max(feeMinor, paidMinor, 1)"
				/>
			</div>

			<h3 class="title is-5 mt-5">
				{{ $t('projectFinance.payment.title') }}
			</h3>

			<form
				class="finance-payment-form"
				@submit.prevent="savePayment"
			>
				<FormField
					id="payment-amount"
					v-model="paymentDraft.amount"
					:label="$t('projectFinance.payment.amount', {currency: currency || '—'})"
					type="text"
					inputmode="decimal"
					placeholder="0.00"
				/>
				<FormField
					id="payment-date"
					v-model="paymentDraft.date"
					:label="$t('projectFinance.payment.date')"
					type="date"
				/>
				<FormField
					id="payment-note"
					v-model="paymentDraft.note"
					:label="$t('projectFinance.payment.note')"
					:placeholder="$t('projectFinance.payment.notePlaceholder')"
					type="text"
					maxlength="250"
				/>
				<div class="finance-payment-form__actions">
					<XButton
						type="submit"
						:icon="paymentDraft.id ? 'check' : 'plus'"
						:loading="savePaymentMutation.isPending.value"
						:disabled="parseMoney(paymentDraft.amount) <= 0"
					>
						{{ paymentDraft.id ? $t('misc.save') : $t('projectFinance.payment.add') }}
					</XButton>
					<XButton
						v-if="paymentDraft.id"
						variant="tertiary"
						@click="resetPaymentDraft"
					>
						{{ $t('misc.cancel') }}
					</XButton>
				</div>
			</form>

			<p
				v-if="paymentsLoaded && payments.length === 0"
				class="has-text-grey is-italic"
			>
				{{ $t('projectFinance.payment.none') }}
			</p>
			<table
				v-else
				class="table is-fullwidth is-striped finance-payments"
			>
				<thead>
					<tr>
						<th>{{ $t('projectFinance.payment.date') }}</th>
						<th>{{ $t('projectFinance.payment.note') }}</th>
						<th class="has-text-right">
							{{ $t('projectFinance.payment.amountShort') }}
						</th>
						<th>
							<span class="is-sr-only">{{ $t('misc.actions') }}</span>
						</th>
					</tr>
				</thead>
				<tbody>
					<tr
						v-for="payment in payments"
						:key="payment.id"
					>
						<td>{{ formatDate(payment.paid_at) }}</td>
						<td>{{ payment.note }}</td>
						<td class="has-text-right">
							{{ formatMoney(payment.amount, currency) }}
						</td>
						<td class="has-text-right finance-payments__actions">
							<BaseButton
								v-tooltip="$t('menu.edit')"
								:aria-label="$t('menu.edit')"
								@click="editPayment(payment)"
							>
								<Icon icon="pen" />
							</BaseButton>
							<BaseButton
								v-tooltip="$t('misc.delete')"
								:aria-label="$t('misc.delete')"
								class="has-text-danger"
								@click="removePayment(payment)"
							>
								<Icon icon="trash-alt" />
							</BaseButton>
						</td>
					</tr>
				</tbody>
			</table>
		</template>
	</CreateEdit>
</template>

<script setup lang="ts">
import {computed, reactive, watch} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {useI18n} from 'vue-i18n'

import BaseButton from '@/components/base/BaseButton.vue'
import CreateEdit from '@/components/misc/CreateEdit.vue'
import FormField from '@/components/input/FormField.vue'
import Message from '@/components/misc/Message.vue'
import XButton from '@/components/input/Button.vue'

import {
	formatMoney,
	minorToInput,
	parseMoney,
	projectFinanceQuery,
	projectPaymentsQuery,
	useDeleteProjectPaymentMutation,
	useSaveProjectFinanceMutation,
	useSaveProjectPaymentMutation,
} from '@/client/queries/projectFinance'
import type {ProjectPaymentResponse} from '@/client/queries/projectFinance'
import {useProject} from '@/composables/useProject'
import {useProjectOverview} from '@/composables/useProjectOverview'
import {useTitle} from '@/composables/useTitle'

const props = defineProps<{
	projectId: number,
}>()

defineOptions({name: 'ProjectSettingsFinance'})

const COMMON_CURRENCIES = ['GBP', 'EUR', 'USD', 'BRL', 'BYN', 'CHF', 'PLN', 'GEL', 'AED']

const {t, locale} = useI18n({useScope: 'global'})

const {project} = useProject(() => props.projectId)
useTitle(() => project.value.title ? t('projectFinance.modalTitle', {project: project.value.title}) : t('projectFinance.money'))

const financeQuery = useQuery(computed(() => projectFinanceQuery(props.projectId)))
const paymentsQuery = useQuery(computed(() => projectPaymentsQuery(props.projectId)))
const {items: overviewItems} = useProjectOverview()

const loadError = computed(() => financeQuery.error.value || paymentsQuery.error.value)
const financeLoaded = computed(() => financeQuery.data.value !== undefined)
const paymentsLoaded = computed(() => paymentsQuery.data.value !== undefined)
const payments = computed(() => paymentsQuery.data.value ?? [])

const knownClients = computed(() =>
	[...new Set(overviewItems.value.map(item => item.client).filter(Boolean))].sort((a, b) => a.localeCompare(b, locale.value)),
)

const draft = reactive({client: '', currency: '', fee: ''})

// Seed the form once per project when its details arrive, so a background
// refetch doesn't overwrite what is being typed.
let seededFor = 0
watch(() => [props.projectId, financeQuery.data.value] as const, ([projectId, finance]) => {
	if (!finance || seededFor === projectId) return
	seededFor = projectId
	draft.client = finance.client
	draft.currency = finance.currency
	draft.fee = finance.fee ? minorToInput(finance.fee) : ''
}, {immediate: true})

const currency = computed(() => draft.currency.trim().toUpperCase())
const feeMinor = computed(() => parseMoney(draft.fee))
const paidMinor = computed(() => payments.value.reduce((sum, payment) => sum + payment.amount, 0))
const owedMinor = computed(() => Math.max(feeMinor.value - paidMinor.value, 0))

const saveFinance = useSaveProjectFinanceMutation()

async function saveDetails() {
	if (!financeLoaded.value || saveFinance.isPending.value) return
	try {
		await saveFinance.mutateAsync({
			project_id: props.projectId,
			client: draft.client.trim(),
			currency: currency.value,
			fee: feeMinor.value,
		})
	} catch {
		return
	}
	draft.currency = currency.value
}

function today(): string {
	const now = new Date()
	const offset = now.getTimezoneOffset() * 60000
	return new Date(now.getTime() - offset).toISOString().slice(0, 10)
}

const paymentDraft = reactive({id: 0, amount: '', date: today(), note: ''})

function resetPaymentDraft() {
	paymentDraft.id = 0
	paymentDraft.amount = ''
	paymentDraft.date = today()
	paymentDraft.note = ''
}

function editPayment(payment: ProjectPaymentResponse) {
	paymentDraft.id = payment.id
	paymentDraft.amount = minorToInput(payment.amount)
	paymentDraft.date = payment.paid_at.slice(0, 10)
	paymentDraft.note = payment.note
}

const savePaymentMutation = useSaveProjectPaymentMutation()
const deletePaymentMutation = useDeleteProjectPaymentMutation()

async function savePayment() {
	const amount = parseMoney(paymentDraft.amount)
	if (amount <= 0 || savePaymentMutation.isPending.value) return
	// Noon UTC keeps the chosen day the same in every time zone.
	const paidAt = new Date(`${paymentDraft.date || today()}T12:00:00Z`).toISOString()
	try {
		await savePaymentMutation.mutateAsync({
			projectId: props.projectId,
			id: paymentDraft.id || undefined,
			amount,
			paidAt,
			note: paymentDraft.note.trim(),
		})
	} catch {
		return
	}
	resetPaymentDraft()
}

function removePayment(payment: ProjectPaymentResponse) {
	if (!window.confirm(t('projectFinance.payment.deleteConfirm', {amount: formatMoney(payment.amount, currency.value)}))) {
		return
	}
	deletePaymentMutation.mutate({projectId: props.projectId, id: payment.id})
	if (paymentDraft.id === payment.id) {
		resetPaymentDraft()
	}
}

function formatDate(value: string): string {
	return new Intl.DateTimeFormat(locale.value, {dateStyle: 'medium', timeZone: 'UTC'}).format(new Date(value))
}
</script>

<style lang="scss" scoped>
.finance-summary {
	display: grid;
	grid-template-columns: repeat(3, 1fr);
	gap: .5rem 1rem;
	padding: 1rem;
	border-radius: $radius;
	background: var(--grey-100);

	> div {
		display: flex;
		flex-direction: column;
	}
}

.finance-summary__label {
	font-size: .8rem;
	color: var(--grey-500);
	text-transform: uppercase;
}

.finance-summary__value {
	font-size: 1.25rem;
	font-weight: 700;
	font-variant-numeric: tabular-nums;
}

.finance-summary__bar {
	grid-column: 1 / -1;
	margin: 0;
	block-size: .5rem;
}

.finance-payment-form {
	display: grid;
	grid-template-columns: 10rem 10rem 1fr auto;
	gap: .75rem;
	align-items: end;
	margin-block-end: 1rem;

	@media screen and (max-width: $tablet) {
		grid-template-columns: 1fr 1fr;
	}
}

.finance-payment-form__actions {
	display: flex;
	gap: .5rem;
	padding-block-end: .75rem;
}

.finance-payments td {
	vertical-align: middle;
	font-variant-numeric: tabular-nums;
}

.finance-payments__actions {
	white-space: nowrap;

	> * + * {
		margin-inline-start: .75rem;
	}
}
</style>
