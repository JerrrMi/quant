package strategy

// Step 为回测与实盘共用的策略单步入口：纯输入、纯输出，无副作用。
// 当前默认委派至 MinimalShortStep；替换完整策略时可改为装配其他纯函数或通过依赖注入挂载实现。
func Step(input AltShortStrategyInput) (AltShortStrategyOutput, error) {
	return MinimalShortStep(input)
}
