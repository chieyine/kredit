export const defaultPersonalOptions = (returns: string) => ({
	channel: 'phone',
	quantity: 1,
	count: 1,
	cadence: 'monthly',
	deposit: '0',
	depositDate: '',
	fulfillment: 'immediate',
	threshold: 50,
	days: 1,
	returns
});
export type PersonalOptions = ReturnType<typeof defaultPersonalOptions>;
