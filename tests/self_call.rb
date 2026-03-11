gauntlet("self method call") do
  class Wallet
    attr_reader :coins

    def initialize(coins)
      @coins = coins
    end

    def value
      @coins * 10
    end

    def summary
      v = self.value
      puts "#{@coins} coins worth #{v}g"
    end
  end

  w = Wallet.new(5)
  w.summary
end

